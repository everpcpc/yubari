package telegram

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

	"yubari/elasticsearch"
)

var (
	page = 5
)

func onSearch(b *Bot, message *tgbotapi.Message) {
	b.setChatAction(message.Chat.ID, "typing")

	idx := getIndex(message)
	q := message.CommandArguments()
	q = strings.TrimSpace(q)
	if q == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "直接发送你要搜索的内容即可。搜索支持 Lucene 语法")
		msg.ReplyToMessageID = message.MessageID
		b.Client.Send(msg)
		return
	}

	exists, err := elasticsearch.CheckIndexExist(b.es, idx)
	if err != nil {
		b.logger.Errorf("check es index exists error: %+v", err)
		return
	}
	if !exists {
		msg := tgbotapi.NewMessage(message.Chat.ID, "还没有启用哦")
		msg.ReplyToMessageID = message.MessageID
		b.Client.Send(msg)
		return
	}
	res, err := elasticsearch.SearchMessage(b.es, idx, q, 0, page)
	if err != nil {
		b.logger.Errorf("es search error: %+v", err)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, buildSearchResponse(b, message.Chat.ID, res, 0))
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

func buildSearchResponse(b *Bot, chatID int64, res *elasticsearch.SearchResponse, from int) string {
	total := res.Hits.Total.Value
	respond := fmt.Sprintf(
		"<code>[%d]</code> results in %s: \n", total, prettyDuration(res.Took))
	for i, hit := range res.Hits.Hits {
		var content string
		if len(hit.Highlight.Content) == 0 {
			content = hit.Source.Content[:15]
		} else {
			content = hit.Highlight.Content[0]
		}
		t := time.Unix(int64(hit.Source.Date/1000), 0)
		author, err := b.GetUserName(chatID, hit.Source.User)
		if err != nil {
			b.logger.Warningf("get username error: %+v", err)
		}
		// TODO:(everpcpc) send link to target message
		// respond += fmt.Sprintf("%d. <a href=\"%d\">%s</a>\n", from+i+1, hit.Source.MessageID, content)
		respond += fmt.Sprintf("%d. <code>[%s]%s</code>: %s\n", from+i+1, t.Format("2006-01-02 15:04:05"), author, content)
	}
	return respond
}

func buildSearchResponseButton(res *elasticsearch.SearchResponse, from int, q string) tgbotapi.InlineKeyboardMarkup {
	total := res.Hits.Total.Value
	encodedQ := base64.StdEncoding.EncodeToString([]byte(q))
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️", fmt.Sprintf("search:%d:%s", max(int64(from-page), 0), encodedQ)),
		tgbotapi.NewInlineKeyboardButtonData("❎", fmt.Sprintf("search:%d:%s", -1, encodedQ)),
		tgbotapi.NewInlineKeyboardButtonData("➡️", fmt.Sprintf("search:%d:%s", max(min(int64(from+page), int64(total-1)), 0), encodedQ)),
	)
	return tgbotapi.NewInlineKeyboardMarkup(row)
}

func onReactionSearch(b *Bot, callbackQuery *tgbotapi.CallbackQuery) {
	b.setChatAction(callbackQuery.Message.Chat.ID, "typing")

	token := strings.Split(callbackQuery.Data, ":")
	if len(token) != 3 {
		b.logger.Errorf("illegal react data: %s", callbackQuery.Data)
		return
	}

	from, err := strconv.ParseInt(token[1], 10, 0)
	if err != nil {
		b.logger.Errorf("illegal react search from: %s", callbackQuery.Data)
		return
	}
	q, err := base64.StdEncoding.DecodeString(token[2])
	if err != nil {
		b.logger.Errorf("illegal react search q: %s", callbackQuery.Data)
		return
	}

	var reply string
	defer func() {
		callbackMsg := tgbotapi.NewCallback(callbackQuery.ID, reply)
		_, err = b.Client.AnswerCallbackQuery(callbackMsg)
		if err != nil {
			b.logger.Errorf("answer callback error: %+v", err)
		}
	}()

	if from < 0 {
		delMsg := tgbotapi.DeleteMessageConfig{
			ChatID:    callbackQuery.Message.Chat.ID,
			MessageID: callbackQuery.Message.MessageID,
		}
		_, err = b.Client.DeleteMessage(delMsg)
		if err != nil {
			reply = fmt.Sprintf("delete error: %+v", err)
		} else {
			reply = "delete OK"
		}
		return
	}

	idx := getIndex(callbackQuery.Message)

	res, err := elasticsearch.SearchMessage(b.es, idx, string(q), int(from), page)
	if err != nil {
		reply = fmt.Sprintf("search error: %+v", err)
		return
	}

	msg := tgbotapi.NewEditMessageText(
		callbackQuery.Message.Chat.ID,
		callbackQuery.Message.MessageID,
		buildSearchResponse(b, callbackQuery.Message.Chat.ID, res, int(from)),
	)
	button := buildSearchResponseButton(res, int(from), string(q))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &button

	_, err = b.Client.Send(msg)
	if err != nil {
		reply = fmt.Sprintf("update result error: %+v", err)
	} else {
		reply = "OK"
	}
}
