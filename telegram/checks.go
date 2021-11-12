package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"golang.org/x/net/html"

	"yubari/elasticsearch"
	"yubari/pixiv"
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
		b.setChatAction(message.Chat.ID, "typing")

		b.redis.Del(key)
		b.logger.Infof("repeat: %s", strconv.Quote(message.Text))
		msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)
		b.Client.Send(msg)
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

	b.setChatAction(message.Chat.ID, "typing")

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
		b.logger.Debugf("download pixiv %d_p%d: %s", id, i, ByteCountIEC(sizes[i]))
		callbackText += fmt.Sprintf("p%d: %s😊 ", i, byteCountBinary(sizes[i]))
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, callbackText)
	msg.ReplyToMessageID = message.MessageID

	_, err := b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func checkSave(b *Bot, message *tgbotapi.Message) {
	if !b.isAuthedChat(message.Chat) {
		return
	}

	idx := getIndex(message)
	exists, err := elasticsearch.CheckIndexExist(b.es, idx)
	if err != nil {
		b.logger.Errorf("%+v", err)
		return
	}
	if !exists {
		err = elasticsearch.CreateIndex(b.es, idx)
		if err != nil {
			b.logger.Errorf("%+v", err)
			return
		}
	}

	article := elasticsearch.Article{
		ID:      message.MessageID,
		User:    message.From.ID,
		Date:    message.Date * 1000,
		Content: html.EscapeString(message.Text),
	}
	err = elasticsearch.StoreMessage(b.es, idx, &article)
	if err != nil {
		b.logger.Errorf("save message error: %+v", err)
	}
}
