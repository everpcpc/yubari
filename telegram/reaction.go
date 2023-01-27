package telegram

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func onReaction(b *Bot, callbackQuery *tgbotapi.CallbackQuery) {
	var callbackText string

	_type, _target, reaction, err := saveReaction(b.redis, callbackQuery.Data, callbackQuery.From.ID)
	if err == nil {
		diss := b.redis.SCard(buildReactionKey(_type, _target, "diss")).Val()
		like := b.redis.SCard(buildReactionKey(_type, _target, "like")).Val()
		if diss-like < 2 {
			msg := tgbotapi.NewEditMessageReplyMarkup(
				callbackQuery.Message.Chat.ID,
				callbackQuery.Message.MessageID,
				buildLikeButton(b.redis, _type, _target),
			)
			_, err = b.Client.Send(msg)
		} else {
			_, err = b.Client.Send(tgbotapi.NewDeleteMessage(
				callbackQuery.Message.Chat.ID,
				callbackQuery.Message.MessageID,
			))
			if err == nil {
				err = b.probate(_type, _target)
			}
		}
	}

	if err != nil {
		b.logger.Debugf("%s", err)
		callbackText = err.Error()
	} else {
		callbackText = reaction + " " + filepath.Base(_target) + "!"
	}

	_, err = b.Client.Request(tgbotapi.NewCallback(callbackQuery.ID, callbackText))
	if err != nil {
		b.logger.Errorf("%s", err)
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
			b.logger.Errorf("%s", err)
			callbackText = "get btd error: " + err.Error()
			break
		}
		newText = fmt.Sprintf("%s recruited:\n%s",
			callbackQuery.From,
			callbackQuery.Message.Text,
		)
		data, err := json.Marshal(DownloadPixiv{
			ChatID:    callbackQuery.Message.Chat.ID,
			MessageID: callbackQuery.Message.MessageID,
			PixivID:   id,
			Text:      newText,
		})
		if err != nil {
			b.logger.Errorf("%s", err)
			callbackText = "marshal message error: " + err.Error()
			break
		}
		err = conn.Use(tgPixivTube)
		if err != nil {
			b.logger.Errorf("%s", err)
			callbackText = "use tube error: " + err.Error()
			break
		}
		_, err = conn.Put(data, 1, 5*time.Second, 10*time.Minute)
		if err != nil {
			callbackText = fmt.Sprintf("queue pixiv error: %s", err)
		} else {
			callbackText = fmt.Sprintf("queued: %d", id)
		}

		newText = fmt.Sprintf("%s recruited:\n%s",
			callbackQuery.From,
			"⏳ "+callbackQuery.Message.Text,
		)

	case "diss":
		newText = fmt.Sprintf("%s expelled pixiv %d", callbackQuery.From, id)

	default:
		b.logger.Errorf("react type error: %s", reaction)
		return
	}

	_, err = b.Client.Request(tgbotapi.NewCallback(callbackQuery.ID, callbackText))
	if err != nil {
		b.logger.Errorf("%s", err)
	}

	updateTextMsg := tgbotapi.NewEditMessageText(
		callbackQuery.Message.Chat.ID,
		callbackQuery.Message.MessageID,
		newText,
	)
	updateTextMsg.DisableWebPagePreview = true
	_, err = b.Client.Send(updateTextMsg)
	if err != nil {
		b.logger.Errorf("error update message text %s", err)
	}
}

func buildReactionData(_type, _target, reaction string) string {
	return _type + ":" + _target + ":" + reaction
}

func buildReactionKey(_type, _target, reaction string) string {
	if strings.Contains(_target, "/") {
		_target = filepath.Base(_target)
	}
	return "reaction_" + buildReactionData(_type, _target, reaction)
}

func buildLikeButton(rds *redis.Client, _type, _target string) tgbotapi.InlineKeyboardMarkup {
	likeCount, _ := rds.SCard(buildReactionKey(_type, _target, "like")).Result()
	dissCount, _ := rds.SCard(buildReactionKey(_type, _target, "diss")).Result()

	likeText := "❤️"
	if likeCount > 0 {
		likeText = likeText + " " + strconv.FormatInt(likeCount, 10)
	}
	dissText := "💔"
	if dissCount > 0 {
		dissText = dissText + " " + strconv.FormatInt(dissCount, 10)
	}

	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(likeText, buildReactionData(_type, _target, "like")),
		tgbotapi.NewInlineKeyboardButtonData("♻️", buildReactionData(_type, _target, "reset")),
		tgbotapi.NewInlineKeyboardButtonData(dissText, buildReactionData(_type, _target, "diss")),
	)
	return tgbotapi.NewInlineKeyboardMarkup(row)
}

func saveReaction(rds *redis.Client, key string, user int64) (_type, _target, reaction string, err error) {
	token := strings.Split(key, ":")
	if len(token) != 3 {
		err = fmt.Errorf("react data error: %s", key)
		return
	}
	_type = token[0]
	_target = token[1]
	reaction = token[2]

	pipe := rds.Pipeline()
	switch reaction {
	case "like":
		likeCount := pipe.SAdd(buildReactionKey(_type, _target, "like"), strconv.FormatInt(user, 10))
		dissCount := pipe.SRem(buildReactionKey(_type, _target, "diss"), strconv.FormatInt(user, 10))
		_, err = pipe.Exec()
		if err == nil {
			if likeCount.Val()+dissCount.Val() == 0 {
				err = fmt.Errorf("not modified")
			}
		}
	case "diss":
		dissCount := pipe.SAdd(buildReactionKey(_type, _target, "diss"), strconv.FormatInt(user, 10))
		likeCount := pipe.SRem(buildReactionKey(_type, _target, "like"), strconv.FormatInt(user, 10))
		_, err = pipe.Exec()
		if err == nil {
			if likeCount.Val()+dissCount.Val() == 0 {
				err = fmt.Errorf("not modified")
			}
		}
	case "reset":
		dissCount := pipe.SRem(buildReactionKey(_type, _target, "diss"), strconv.FormatInt(user, 10))
		likeCount := pipe.SRem(buildReactionKey(_type, _target, "like"), strconv.FormatInt(user, 10))
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
