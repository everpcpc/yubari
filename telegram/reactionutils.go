package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

	"github.com/go-redis/redis"
)

func buildReactionData(_type, _id, reaction string) string {
	return _type + ":" + _id + ":" + reaction
}
func buildReactionKey(_type, _id, reaction string) string {
	return "reaction_" + buildReactionData(_type, _id, reaction)
}

func buildInlineKeyboardMarkup(rds *redis.Client, _type, _id string) tgbotapi.InlineKeyboardMarkup {

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
