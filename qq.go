package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	bt "github.com/ikool-cn/gobeanstalk-connection-pool"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	qqBot *QQBot
	reCQ  = regexp.MustCompile(`\[CQ:(image|at|face),(file|qq|id)\=([\w\.]+)\]`)
)

// QQFace ...
type QQFace int

// QQAt ...
type QQAt struct {
	qq string
}

// QQImage ...
type QQImage struct {
	file string
}

// QQBot ...
type QQBot struct {
	ID     string
	Config *QQConfig
	Client *bt.Pool
	RecvQ  string
	SendQ  string
}

// NewQQBot ...
func NewQQBot(cfg *Config) *QQBot {
	q := &QQBot{ID: cfg.QQ.BotID, Config: cfg.QQ}
	q.Client = &bt.Pool{
		Dial: func() (*bt.Conn, error) {
			return bt.Dial(cfg.BeanstalkAddr)
		},
		MaxIdle:     10,
		MaxActive:   100,
		IdleTimeout: 60 * time.Second,
		MaxLifetime: 180 * time.Second,
		Wait:        true,
	}

	q.RecvQ = q.ID + "(o)"
	q.SendQ = q.ID + "(i)"
	return q
}

// String generate code string for qq face
func (q QQFace) String() string {
	return fmt.Sprintf("[CQ:face,id=%d]", q)
}

// String generate code string for qq image
func (q QQImage) String() string {
	return fmt.Sprintf("[CQ:image,file=%s]", q.file)
}

// String generate code string for qq qt msg
func (q QQAt) String() string {
	return fmt.Sprintf("[CQ:at,qq=%s]", q.qq)
}

func (q *QQBot) send(msg []byte) {
	// wait longer with more errors
	var (
		conn *bt.Conn
		err  error
	)
	for i := 1; ; i++ {
		conn, err = q.Client.Get()
		if err == nil {
			break
		}
		time.Sleep(time.Duration(i) * time.Second)
		if i > q.Config.SendMaxRetry {
			logger.Error("Send failed:", string(msg))
			return
		}
	}
	conn.Use(q.SendQ)
	_, err = conn.Put(msg, 1, 0, time.Minute)
	if err != nil {
		logger.Error(err)
		return
	}

	q.Client.Release(conn, false)
	return
}

// SendGroupMsg ...
func (q *QQBot) SendGroupMsg(msg string) {
	logger.Infof(strconv.Quote(msg))
	fullMsg, err := formMsg("sendGroupMsg", q.Config.GroupID, msg)
	if err != nil {
		logger.Error(err)
		return
	}
	go q.send(fullMsg)
}

// SendPics ...
func (q *QQBot) SendPics(fn func(string), url string) {
	fileName, err := downloadFile(url, q.Config.ImgPath)
	if err != nil {
		return
	}
	fn(QQImage{fileName}.String())
}

// SendPrivateMsg ...
func (q *QQBot) SendPrivateMsg(qq string, msg string) {
	logger.Infof("TO:" + qq + ":" + strconv.Quote(msg))
	fullMsg, err := formMsg("sendPrivateMsg", qq, msg)
	if err != nil {
		logger.Error(err)
	} else {
		go q.send(fullMsg)
	}
}

//SendSelfMsg ...
func (q *QQBot) SendSelfMsg(msg string) {
	q.SendPrivateMsg(q.Config.SelfID, msg)
}

// CheckMention ...
func (q *QQBot) CheckMention(msg string) bool {
	for _, s := range q.Config.SelfNames {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// NoticeMention ...
func (q *QQBot) NoticeMention(msg string, group string) {
	if !q.CheckMention(msg) {
		return
	}
	key := q.Config.SelfID + "_mention"
	exists, err := redisClient.Expire(key, 10*time.Minute).Result()
	if err != nil {
		logger.Error(err)
		return
	}
	if exists {
		logger.Notice("Called in last 10min")
	} else {
		_, err := redisClient.Set(key, 0, 10*time.Minute).Result()
		if err != nil {
			logger.Error(err)
			return
		}
		q.SendGroupMsg("呀呀呀，召唤一号机" + QQAt{q.Config.SelfID}.String())
	}
}

// CheckRepeat ...
func (q *QQBot) CheckRepeat(msg string, group string) {
	key := group + "_last"
	flattendMsg := strings.TrimSpace(msg)
	defer redisClient.LTrim(key, 0, 10)
	defer redisClient.LPush(key, flattendMsg)
	lastMsgs, err := redisClient.LRange(key, 0, 5).Result()
	if err != nil {
		logger.Error(err)
		return
	}
	i := 0
	for _, s := range lastMsgs {
		if s == flattendMsg {
			i++
		}
	}
	if i > 1 {
		redisClient.Del(key)
		logger.Infof("Repeat: %s", strconv.Quote(msg))
		q.SendGroupMsg(msg)
	}
}

// CheckAt ...
func (q *QQBot) CheckAt(msg string) bool {
	ss := reCQ.FindAllStringSubmatch(msg, -1)
	for _, s := range ss {
		if s[1] == "at" {
			if s[3] == q.Config.BotID {
				return true
			}
		}
	}
	return false
}

// Poll reserve msg from beanstalkd
func (q *QQBot) Poll(messages chan map[string]string) {
	for {
		conn, err := q.Client.Get()
		if err != nil {
			logger.Error(err)
			time.Sleep(3 * time.Second)
			continue
		}
		conn.Watch(q.RecvQ)
		job, err := conn.Reserve()
		if err != nil {
			logger.Warning(err)
			time.Sleep(3 * time.Second)
			continue
		}
		logger.Debug("get msg:", string(job.Body))
		body := strings.Split(string(job.Body), " ")
		ret := make(map[string]string)
		switch body[0] {
		case "eventPrivateMsg":
			ret["event"] = "PrivateMsg"
			ret["subtype"] = body[1]
			ret["time"] = body[2]
			ret["qq"] = body[3]
			ret["msg"], err = decodeMsg(body[4])
			if err != nil {
				if body[4] == "0" {
					logger.Warning("empty msg")
					err = conn.Bury(job.ID, 0)
				}
				if err != nil {
					logger.Error(err)
				}
				time.Sleep(3 * time.Second)
				continue
			}
		case "eventGroupMsg":
			ret["event"] = "GroupMsg"
			ret["subtype"] = body[1]
			ret["time"] = body[2]
			ret["group"] = body[3]
			ret["qq"] = body[4]
			ret["anonymous"] = body[5]
			ret["msg"], err = decodeMsg(body[6])
			if err != nil {
				if body[6] == "0" {
					logger.Warning("empty msg")
					err = conn.Bury(job.ID, 0)
				}
				if err != nil {
					logger.Error(err)
				}
				time.Sleep(3 * time.Second)
				continue
			}
		default:
			err = conn.Bury(job.ID, 0)
			if err != nil {
				logger.Error(err)
				time.Sleep(3 * time.Second)
			}
			continue
		}
		messages <- ret
		err = conn.Delete(job.ID)
		if err != nil {
			logger.Error(err)
			time.Sleep(3 * time.Second)
		}
		q.Client.Release(conn, false)
	}
}

func formMsg(t string, to string, msg string) ([]byte, error) {
	gb18030Msg, err := Utf8ToGb18030([]byte(msg))
	if err != nil {
		return nil, err
	}
	base64Msg := base64.StdEncoding.EncodeToString(gb18030Msg)
	return bytes.Join([][]byte{[]byte(t), []byte(to), []byte(base64Msg)}, []byte(" ")), nil
}

func decodeMsg(msg string) (string, error) {
	gb18030Msg, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return "", err
	}
	utf8Msg, err := Gb18030ToUtf8(gb18030Msg)
	if err != nil {
		return "", err
	}
	return string(utf8Msg), nil
}

func qqWatch(messages chan map[string]string) {
	groupIgnore := make(map[string]struct{})
	for _, q := range qqBot.Config.QQGroupIgnore {
		groupIgnore[q] = struct{}{}
	}
	privateIgnore := make(map[string]struct{})
	for _, q := range qqBot.Config.QQPrivateIgnore {
		privateIgnore[q] = struct{}{}
	}

	for msg := range messages {
		switch msg["event"] {
		case "PrivateMsg":
			if _, ok := privateIgnore[msg["qq"]]; ok {
				// logger.Debugf("Ignore [%s]:{%s}", msg["qq"], strconv.Quote(msg["msg"]))
				continue
			}
			logger.Infof("[%s]:{%s}", msg["qq"], strconv.Quote(msg["msg"]))
		case "GroupMsg":
			if _, ok := groupIgnore[msg["qq"]]; ok {
				logger.Debugf("Ignore (%s)[%s]:{%s}", msg["group"], msg["qq"], strconv.Quote(msg["msg"]))
				continue
			}
			go qqBot.NoticeMention(msg["msg"], msg["group"])
			go qqBot.CheckRepeat(msg["msg"], msg["group"])
			if qqBot.CheckAt(msg["msg"]) {
				cleanMsg := reCQ.ReplaceAllString(msg["msg"], "")
				logger.Infof("at: (%s)[%s]:{%s}", msg["group"], msg["qq"], strconv.Quote(cleanMsg))
				qqBot.SendGroupMsg("不要随便 @ 人家啦>_<")
			} else {
				logger.Infof("(%s)[%s]:{%s}", msg["group"], msg["qq"], strconv.Quote(msg["msg"]))
			}
		default:
			logger.Info(msg)
		}
	}
}
