package telegram

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

	"yubari/pixiv"
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
				buildLikeButton(b.redis, _type, _id),
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

func buildReactionData(_type, _id, reaction string) string {
	return _type + ":" + _id + ":" + reaction
}
func buildReactionKey(_type, _id, reaction string) string {
	return "reaction_" + buildReactionData(_type, _id, reaction)
}

func buildLikeButton(rds *redis.Client, _type, _id string) tgbotapi.InlineKeyboardMarkup {

	likeCount, _ := rds.SCard(buildReactionKey(_type, _id, "like")).Result()
	dissCount, _ := rds.SCard(buildReactionKey(_type, _id, "diss")).Result()

	likeText := "❤️"
	if likeCount > 0 {
		likeText = likeText + " " + strconv.FormatInt(likeCount, 10)
	}
	dissText := "💔"
	if dissCount > 0 {
		dissText = dissText + " " + strconv.FormatInt(dissCount, 10)
	}

	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(likeText, buildReactionData(_type, _id, "like")),
		tgbotapi.NewInlineKeyboardButtonData(dissText, buildReactionData(_type, _id, "diss")),
	)
	return tgbotapi.NewInlineKeyboardMarkup(row)
}

func saveReaction(rds *redis.Client, key string, user int) (_type, _id, reaction string, err error) {
	token := strings.Split(key, ":")
	if len(token) != 3 {
		err = fmt.Errorf("react data error: %s", key)
		return
	}
	_type = token[0]
	_id = token[1]
	reaction = token[2]

	pipe := rds.Pipeline()
	switch reaction {
	case "like":
		likeCount := pipe.SAdd(buildReactionKey(_type, _id, "like"), strconv.Itoa(user))
		dissCount := pipe.SRem(buildReactionKey(_type, _id, "diss"), strconv.Itoa(user))
		_, err = pipe.Exec()
		if err == nil {
			if likeCount.Val()+dissCount.Val() == 0 {
				err = fmt.Errorf("not modified")
			}
		}
	case "diss":
		dissCount := pipe.SAdd(buildReactionKey(_type, _id, "diss"), strconv.Itoa(user))
		likeCount := pipe.SRem(buildReactionKey(_type, _id, "like"), strconv.Itoa(user))
		_, err = pipe.Exec()
		if err == nil {
			if likeCount.Val()+dissCount.Val() == 0 {
				err = fmt.Errorf("not modified")
			}
		}
	default:
		err = fmt.Errorf("react type error: %s", key)
	}
	return
}
