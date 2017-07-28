package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	bt "github.com/ikool-cn/gobeanstalk-connection-pool"
	"strings"
	"time"
	// "regexp"
)

var (
	qqBot *QQBot
)

type QQface struct {
	Id int
}

type QQBot struct {
	Id    string
	Cfg   *Config
	Pool  *bt.Pool
	RecvQ string
	SendQ string
}

func NewQQBot(cfg *Config) *QQBot {
	q := &QQBot{Id: cfg.QQBot, Cfg: cfg}
	q.Pool = &bt.Pool{
		Dial: func() (*bt.Conn, error) {
			return bt.Dial(cfg.BeanstalkAddr)
		},
		MaxIdle:     10,
		MaxActive:   100,
		IdleTimeout: 60 * time.Second,
		MaxLifetime: 180 * time.Second,
		Wait:        true,
	}

	q.RecvQ = fmt.Sprintf("%s(o)", q.Id)
	q.SendQ = fmt.Sprintf("%s(i)", q.Id)
	return q
}

func (q *QQface) String() string {
	return fmt.Sprintf("[CQ:face,id=%d]", q.Id)
}

func (q *QQBot) send(msg []byte) {
	// wait longer with more errors
	var (
		conn *bt.Conn
		err  error
	)
	for i := 1; ; i++ {
		conn, err = q.Pool.Get()
		if err == nil {
			break
		}
		time.Sleep(time.Duration(i) * time.Second)
		if i > q.Cfg.QQSendMaxRetry {
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

	q.Pool.Release(conn, false)
	return
}

func (q *QQBot) SendGroupMsg(msg string) {
	fullMsg, err := formMsg("sendGroupMsg", q.Cfg.QQGroup, msg)
	if err != nil {
		logger.Error(err)
		return
	}
	go q.send(fullMsg)
}

func (q *QQBot) SendPrivateMsg(qq string, msg string) {
	fullMsg, err := formMsg("sendPrivateMsg", qq, msg)
	if err != nil {
		logger.Error(err)
	} else {
		go q.send(fullMsg)
	}
}

func (q *QQBot) SendSelfMsg(msg string) {
	q.SendPrivateMsg(q.Cfg.QQSelf, msg)
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

func (q *QQBot) Poll(messages chan map[string]string) {
	for i := 1; ; i++ {
		conn, err := q.Pool.Get()
		if err != nil {
			logger.Error(err)
			time.Sleep(time.Duration(i) * time.Second)
			continue
		}
		conn.Watch(q.RecvQ)
		job, err := conn.Reserve()
		if err != nil {
			logger.Warning(err)
			time.Sleep(time.Duration(i) * time.Second)
			continue
		}
		i = 1
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
				logger.Error(err)
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
				logger.Error(err)
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
		q.Pool.Release(conn, false)
	}
}

/*
	data := []byte(`{"/laugh": 12, "/cry": 2}`)
	var objmap map[string]*json.RawMessage
	err := json.Unmarshal(data, &objmap)
	if err != nil {
	fmt.Println(err)
		return
	}
	faceId, err := strconv.Atoi(string(*objmap["/laugh"]))
	if err != nil {
		fmt.Println(err)
		return
	}
	face := QQface{faceId}
	fmt.Println(face.String())
*/
