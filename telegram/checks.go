package telegram

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/everpcpc/yubari/pixiv"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func checkRepeat(b *Bot, message *tgbotapi.Message) {
	key := "tg_last_" + strconv.FormatInt(message.Chat.ID, 10)
	flattendMsg := strings.TrimSpace(message.Text)
	defer b.redis.LTrim(key, 0, 10)
	defer b.redis.LPush(key, flattendMsg)

	lastMsgs, err := b.redis.LRange(key, 0, 6).Result()
	if err != nil {
		b.logger.Errorf("%+v", err)
		return
	}
	i := 0
	for _, s := range lastMsgs {
		if s == flattendMsg {
			i++
		}
	}
	if i > 1 {
		b.redis.Del(key)
		b.logger.Infof("repeat: %s", strconv.Quote(message.Text))
		msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)
		go b.Client.Send(msg)
	}
}

func checkPixiv(b *Bot, message *tgbotapi.Message) {
	if !b.isAuthedChat(message.Chat) {
		return
	}
	id := pixiv.ParseURL(message.Text)
	if id == 0 {
		return
	}
	var callbackText string
	sizes, errs := pixiv.Download(id, b.PixivPath)
	for i := range sizes {
		if errs[i] != nil {
			callbackText += fmt.Sprintf("p%d: error😕 ", i)
			continue
		}
		if sizes[i] == 0 {
			callbackText += fmt.Sprintf("p%d: exists😋 ", i)
			continue
		}
		b.logger.Debugf("download pixiv %d_p%d: %d bytes", id, i, sizes[i])
		callbackText += fmt.Sprintf("p%d: %s😊 ", i, byteCountBinary(sizes[i]))
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, callbackText)
	msg.ReplyToMessageID = message.MessageID

	_, err := b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}
