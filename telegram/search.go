package telegram

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/everpcpc/yubari/elasticsearch"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	page = 5
)

func onSearch(b *Bot, message *tgbotapi.Message) {
	idx := getIndex(message)
	q := message.CommandArguments()

	exists, err := elasticsearch.CheckIndexExist(b.es, idx)
	if err != nil {
		b.logger.Errorf("check es index exists error: %+v", err)
		return
	}
	if !exists {
		msg := tgbotapi.NewMessage(message.Chat.ID, "还没有启用哦")
		msg.ReplyToMessageID = message.MessageID
		b.Client.Send(msg)
	}
	res, err := elasticsearch.SearchMessage(b.es, idx, q, 0, page)
	if err != nil {
		b.logger.Errorf("es search error: %+v", err)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, buildSearchResponse(res, 0))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = buildSearchResponseButton(res, 0, q)

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func getIndex(message *tgbotapi.Message) string {
	return fmt.Sprintf("%s-%d", message.Chat.Type, message.Chat.ID)
}

func buildSearchResponse(res *elasticsearch.SearchResponse, from int) string {
	total := res.Hits.Total.Value
	respond := fmt.Sprintf("(%d) results: \n", total)
	for i, hit := range res.Hits.Hits {
		var content string
		if len(hit.Highlight.Content) == 0 {
			content = hit.Source.Content[:15]
		} else {
			content = hit.Highlight.Content[0]
		}
		respond += fmt.Sprintf("%d. <a href=\"%d\">%s</a>\n", from+i+1, hit.Source.MessageID, content)
	}
	respond += fmt.Sprintf("duration %.3fs", float64(res.Took)/1000)
	return respond
}

func buildSearchResponseButton(res *elasticsearch.SearchResponse, from int, q string) tgbotapi.InlineKeyboardMarkup {
	total := res.Hits.Total.Value
	if total == 0 {
		total = 1
	}
	encodedQ := base64.StdEncoding.EncodeToString([]byte(q))
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️", fmt.Sprintf("search:%d:%s", max(int64(from-5), 0), encodedQ)),
		tgbotapi.NewInlineKeyboardButtonData("❎", fmt.Sprintf("search:%d:%s", -1, encodedQ)),
		tgbotapi.NewInlineKeyboardButtonData("➡️", fmt.Sprintf("search:%d:%s", min(int64(from+5), int64(total-1)), encodedQ)),
	)
	return tgbotapi.NewInlineKeyboardMarkup(row)
}

func onReactionSearch(b *Bot, callbackQuery *tgbotapi.CallbackQuery) {
	token := strings.Split(callbackQuery.Data, ":")
	if len(token) != 3 {
		b.logger.Errorf("react data error: %s", callbackQuery.Data)
		return
	}

	from, err := strconv.ParseInt(token[1], 10, 0)
	if err != nil {
		b.logger.Errorf("react search from error: %s", callbackQuery.Data)
	}
	if from < 0 {
		delMsg := tgbotapi.DeleteMessageConfig{
			ChatID:    callbackQuery.Message.Chat.ID,
			MessageID: callbackQuery.Message.MessageID,
		}
		_, err = b.Client.DeleteMessage(delMsg)
		if err != nil {
			b.logger.Errorf("delete search error: %s", callbackQuery.Data)
		}
		return
	}

	q, err := base64.StdEncoding.DecodeString(token[2])
	if err != nil {
		b.logger.Errorf("react search q error: %s", callbackQuery.Data)
		return
	}

	idx := getIndex(callbackQuery.Message)

	res, err := elasticsearch.SearchMessage(b.es, idx, string(q), int(from), page)
	if err != nil {
		b.logger.Errorf("es search error: %+v", err)
		return
	}

	msg := tgbotapi.NewEditMessageText(
		callbackQuery.Message.Chat.ID,
		callbackQuery.Message.MessageID,
		buildSearchResponse(res, int(from)),
	)
	button := buildSearchResponseButton(res, int(from), string(q))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &button

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("search reaction reply error: %+v", err)
	}

	callbackMsg := tgbotapi.NewCallback(callbackQuery.ID, "OK")
	_, err = b.Client.AnswerCallbackQuery(callbackMsg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}
