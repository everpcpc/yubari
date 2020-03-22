package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

	"github.com/everpcpc/yubari/pixiv"
)

func onReaction(b *Bot, callbackQuery *tgbotapi.CallbackQuery) {
	var callbackText string

	_type, _id, reaction, err := saveReaction(b.redis, callbackQuery.Data, callbackQuery.From.ID)
	if err == nil {
		diss := b.redis.SCard(buildReactionKey(_type, _id, "diss")).Val()
		like := b.redis.SCard(buildReactionKey(_type, _id, "like")).Val()
		if diss-like < 2 {
			msg := tgbotapi.NewEditMessageReplyMarkup(
				callbackQuery.Message.Chat.ID,
				callbackQuery.Message.MessageID,
				buildInlineKeyboardMarkup(b.redis, _type, _id),
			)
			_, err = b.Client.Send(msg)
		} else {
			delMsg := tgbotapi.DeleteMessageConfig{
				ChatID:    callbackQuery.Message.Chat.ID,
				MessageID: callbackQuery.Message.MessageID,
			}
			_, err = b.Client.DeleteMessage(delMsg)
			if err == nil {
				err = b.probate(_type, _id)
			}
		}
	}

	if err != nil {
		b.logger.Debugf("%+v", err)
		callbackText = err.Error()
	} else {
		callbackText = reaction + " " + _id + "!"
	}

	callbackMsg := tgbotapi.NewCallback(callbackQuery.ID, callbackText)
	_, err = b.Client.AnswerCallbackQuery(callbackMsg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func onReactionSelf(b *Bot, callbackQuery *tgbotapi.CallbackQuery) {

	var callbackText string

	token := strings.Split(callbackQuery.Data, ":")
	if len(token) != 3 {
		b.logger.Errorf("react data error: %s", callbackQuery.Data)
		return
	}
	_id := token[1]
	reaction := token[2]
	switch reaction {
	case "like":
		id, err := strconv.ParseUint(_id, 10, 0)
		if err != nil {
			callbackText = "failed parsing pixiv id"
			break
		}
		sizes, errs := pixiv.Download(id, b.PixivPath)
		for i := range sizes {
			if errs[i] != nil {
				callbackText += fmt.Sprintf("p%d: error;", i)
				continue
			}
			if sizes[i] == 0 {
				callbackText += fmt.Sprintf("p%d: exists;", i)
				continue
			}
			b.logger.Debugf("download pixiv %d_p%d: %d bytes", id, i, sizes[i])
			callbackText += fmt.Sprintf("p%d: %s;", i, byteCountBinary(sizes[i]))
		}

	case "diss":
	default:
		callbackText = fmt.Sprintf("react type error: %s", reaction)
	}

	delMsg := tgbotapi.DeleteMessageConfig{
		ChatID:    callbackQuery.Message.Chat.ID,
		MessageID: callbackQuery.Message.MessageID,
	}
	_, err := b.Client.DeleteMessage(delMsg)
	if err != nil {
		b.logger.Errorf("failed deleting msg: %+v", err)
	}

	callbackMsg := tgbotapi.NewCallback(callbackQuery.ID, callbackText)
	_, err = b.Client.AnswerCallbackQuery(callbackMsg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}
