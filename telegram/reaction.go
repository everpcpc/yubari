package telegram

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
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

func onReactionCandidate(b *Bot, callbackQuery *tgbotapi.CallbackQuery) {
	token := strings.Split(callbackQuery.Data, ":")
	if len(token) != 3 {
		b.logger.Errorf("react data error: %s", callbackQuery.Data)
		return
	}

	_id := token[1]
	id, err := strconv.ParseUint(_id, 10, 0)
	if err != nil {
		b.logger.Errorf("failed parsing pixiv id (%s): %s", err, callbackQuery.Data)
		return
	}
	reaction := token[2]

	var newText, callbackText string
	switch reaction {
	case "like":
		conn, err := b.Queue.Get()
		if err != nil {
			b.logger.Errorf("%+v", err)
			callbackText = "get btd error: " + err.Error()
			break
		}
		data, err := json.Marshal(DownloadPixiv{
			ChatID:    callbackQuery.Message.Chat.ID,
			MessageID: callbackQuery.Message.MessageID,
			PixivID:   id,
		})
		if err != nil {
			b.logger.Errorf("%+v", err)
			callbackText = "marshal message error: " + err.Error()
			break
		}
		err = conn.Use(tgPixivTube)
		if err != nil {
			b.logger.Errorf("%+v", err)
			callbackText = "use tube error: " + err.Error()
			break
		}
		_, err = conn.Put(data, 1, 0, 10*time.Minute)
		if err != nil {
			callbackText = fmt.Sprintf("queue pixiv error: %s", err)
		} else {
			callbackText = fmt.Sprintf("queued: %d", id)
		}

		newText = fmt.Sprintf("%s recruited pixiv:\n%s",
			callbackQuery.From,
			callbackQuery.Message.Text,
		)

	case "diss":
		delMsg := tgbotapi.DeleteMessageConfig{
			ChatID:    callbackQuery.Message.Chat.ID,
			MessageID: callbackQuery.Message.MessageID,
		}
		_, err := b.Client.DeleteMessage(delMsg)
		if err != nil {
			b.logger.Errorf("failed deleting msg: %+v", err)
		}

		newText = fmt.Sprintf("%s persuaded pixiv %d to quit", callbackQuery.From, id)

	default:
		callbackText = fmt.Sprintf("react type error: %s", reaction)
	}

	callbackMsg := tgbotapi.NewCallback(callbackQuery.ID, callbackText)
	_, err = b.Client.AnswerCallbackQuery(callbackMsg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}

	delBtnMsg := tgbotapi.NewEditMessageReplyMarkup(
		callbackQuery.Message.Chat.ID,
		callbackQuery.Message.MessageID,
		tgbotapi.NewInlineKeyboardMarkup(),
	)
	_, err = b.Client.Send(delBtnMsg)
	if err != nil {
		b.logger.Errorf("error delete inline markup %s", err)
	}

	updateTextMsg := tgbotapi.NewEditMessageText(
		callbackQuery.Message.Chat.ID,
		callbackQuery.Message.MessageID,
		newText,
	)
	_, err = b.Client.Send(updateTextMsg)
	if err != nil {
		b.logger.Errorf("error update message text %s", err)
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
		tgbotapi.NewInlineKeyboardButtonData("♻️", buildReactionData(_type, _id, "reset")),
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
	case "reset":
		dissCount := pipe.SRem(buildReactionKey(_type, _id, "diss"), strconv.Itoa(user))
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
