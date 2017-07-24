package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	btd "github.com/kr/beanstalk"
	"time"
	// "regexp"
)

var (
	qqGroup = ""
	selfQQ  = ""
)

type QQface struct {
	Id int
}

type QQBot struct {
	Id     int
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
	q.SendQ = &btd.Tube{Conn: client, Name: fmt.Sprintf("%d(i)", q.Id)}
	return nil
}

func (q *QQBot) send(msg []byte) error {
	_, err := q.SendQ.Put(msg, 1, 0, time.Minute)
	return err
}

func (q *QQBot) SendGroupMsg(msg string) error {
	fullMsg, err := formMsg("sendGroupMsg", qqGroup, msg)
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
	return q.SendPrivateMsg(selfQQ, msg)
}

func formMsg(t string, to string, msg string) ([]byte, error) {
	gb18030Msg, err := Utf8ToGb18030([]byte(msg))
	if err != nil {
		return nil, err
	}
	base64Msg := base64.StdEncoding.EncodeToString(gb18030Msg)
	return bytes.Join([][]byte{[]byte(t), []byte(to), []byte(base64Msg)}, []byte(" ")), nil
}
