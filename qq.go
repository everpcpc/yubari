package yubari

import (
	"fmt"
	// btd "github.com/kr/beanstalk"
	// "regexp"
)

type QQface struct {
	Id int
}

type QQClient struct {
}

func (q *QQface) String() string {
	return fmt.Sprintf("[CQ:face,id=%d]", q.Id)
}

func (q *QQClient) Send(msg string) {
}
