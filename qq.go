package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	bt "github.com/kr/beanstalk"
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
	Id     string
	Cfg    *Config
	Client *bt.Conn
	SendQ  *bt.Tube
	RecvQ  *bt.TubeSet
}

func NewQQBot(cfg *Config) (*QQBot, error) {
	q := &QQBot{Id: cfg.QQBot, Cfg: cfg}
	client, err := bt.Dial("tcp", cfg.BeanstalkAddr)
	if err != nil {
		return q, err
	}
	q.Client = client
	q.SendQ = &bt.Tube{Conn: client, Name: fmt.Sprintf("%s(i)", q.Id)}
	q.RecvQ = bt.NewTubeSet(client, fmt.Sprintf("%s(o)", q.Id))
	return q, nil
}

func (q *QQface) String() string {
	return fmt.Sprintf("[CQ:face,id=%d]", q.Id)
}

func (q *QQBot) send(msg []byte) error {
	_, err := q.SendQ.Put(msg, 1, 0, time.Minute)
	return err
}

func (q *QQBot) SendGroupMsg(msg string) error {
	fullMsg, err := formMsg("sendGroupMsg", q.Cfg.QQGroup, msg)
	if err != nil {
		return err
	}
	return q.send(fullMsg)
}

func (q *QQBot) SendPrivateMsg(qq string, msg string) error {
	fullMsg, err := formMsg("sendPrivateMsg", qq, msg)
	if err != nil {
		return err
	}
	return q.send(fullMsg)
}

func (q *QQBot) SendSelfMsg(msg string) error {
	return q.SendPrivateMsg(q.Cfg.QQSelf, msg)
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
	for true {
		id, body_, err := q.RecvQ.Reserve(1 * time.Hour)
		if err != nil {
			logger.Warning(err)
			continue
		}
		body := strings.Split(string(body_), " ")
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
				continue
			}
		default:
			err = q.Client.Bury(id, 0)
			if err != nil {
				logger.Error(err)
			}
			continue
		}
		messages <- ret
		err = q.Client.Delete(id)
		if err != nil {
			logger.Error(err)
		}
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
