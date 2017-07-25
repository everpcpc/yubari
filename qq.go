package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	btd "github.com/kr/beanstalk"
	"strings"
	"time"
	// "regexp"
)

type QQface struct {
	Id int
}

type QQBot struct {
	Id     string
	Cfg    *Config
	Client *btd.Conn
	SendQ  *btd.Tube
	RecvQ  *btd.TubeSet
}

func (q *QQface) String() string {
	return fmt.Sprintf("[CQ:face,id=%d]", q.Id)
}

func (q *QQBot) Connect(addr string) error {
	client, err := btd.Dial("tcp", addr)
	if err != nil {
		return err
	}
	q.Client = client
	q.SendQ = &btd.Tube{Conn: client, Name: fmt.Sprintf("%s(i)", q.Id)}
	q.RecvQ = btd.NewTubeSet(client, fmt.Sprintf("%s(o)", q.Id))
	return nil
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
	return q.SendPrivateMsg(q.Cfg.SelfQQ, msg)
}

func formMsg(t string, to string, msg string) ([]byte, error) {
	gb18030Msg, err := Utf8ToGb18030([]byte(msg))
	if err != nil {
		return nil, err
	}
	base64Msg := base64.StdEncoding.EncodeToString(gb18030Msg)
	return bytes.Join([][]byte{[]byte(t), []byte(to), []byte(base64Msg)}, []byte(" ")), nil
}

func (q *QQBot) Poll() {
	for true {
		id, body_, err := q.RecvQ.Reserve(1 * time.Hour)
		if err != nil {
			Logger.Warning(err)
			continue
		}
		body := strings.Split(string(body_), " ")
		ret := make(map[string]string)
		switch body[0] {
		case "eventPrivateMsg":
			ret["event"] = "PrivateMsg"
		case "eventGroupMsg":
			ret["event"] = "GroupMsg"
		default:
			err = q.Client.Bury(id, 0)
		}
	}
}
