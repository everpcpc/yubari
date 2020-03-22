package qq

import "fmt"

type Config struct {
	SelfID          string   `json:"selfID"`
	BotID           string   `json:"botID"`
	QQPrivateIgnore []string `json:"qqPrivateIgnore"`
	QQGroupIgnore   []string `json:"qqGroupIgnore"`
	SelfNames       []string `json:"selfNames"`
	SendMaxRetry    int      `json:"sendMaxRetry"`
	ImgPath         string   `json:"imgPath"`
}

type QQFace int

// String generate code string for qq face
func (q QQFace) String() string {
	return fmt.Sprintf("[CQ:face,id=%d]", q)
}

type QQAt struct {
	qq string
}

// String generate code string for qq qt msg
func (q QQAt) String() string {
	return fmt.Sprintf("[CQ:at,qq=%s]", q.qq)
}

type QQImage struct {
	file string
}

// String generate code string for qq image
func (q QQImage) String() string {
	return fmt.Sprintf("[CQ:image,file=%s]", q.file)
}
