package telegram

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
	"yubari/meili"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	meilisearch "github.com/meilisearch/meilisearch-go"
)

var (
	page = int64(5)
)

func onSearch(b *Bot, message *tgbotapi.Message) {
	b.setChatAction(message.Chat.ID, "typing")

	q := message.CommandArguments()
	q = strings.TrimSpace(q)
	if q == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "直接发送你要搜索的内容即可")
		msg.ReplyToMessageID = message.MessageID
		b.Client.Send(msg)
		return
	}

	idx := b.getIndex(message)
	res, err := idx.Search(q, &meilisearch.SearchRequest{
		Limit: page,
	})
	if err != nil {
		b.logger.Errorf("es search error: %+v", err)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, buildSearchResponse(b, message.Chat.ID, res, 0))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = buildSearchResponseButton(res.NbHits, 0, q)

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func buildSearchResponse(b *Bot, chatID int64, res *meilisearch.SearchResponse, from int) string {
	total := res.NbHits
	respond := fmt.Sprintf(
		"<code>[%d]</code> results in %s: \n", total, prettyDuration(res.ProcessingTimeMs))
	hits, err := meili.DecodeArticles(res.Hits)
	if err != nil {
		b.logger.Error("search error: %+v", err)
	}
	for i, hit := range hits {
		t := time.Unix(hit.Date, 0)
		author, err := b.GetUserName(chatID, int(hit.User))
		if err != nil {
			b.logger.Warningf("get username error: %+v", err)
		}
		if hit.ID > 0 {
			respond += fmt.Sprintf("%d. <a href=\"tg://privatepost?channel=%d&post=%d\">[%s]</a><code>%s</code>: %s\n",
				from+i+1, getSuperGroupChatID(chatID), hit.ID, t.Format("2006-01-02 15:04:05"), author, hit.Content)
		} else {
			respond += fmt.Sprintf("%d. [%s]<code>%s</code>: %s\n",
				from+i+1, t.Format("2006-01-02 15:04:05"), author, hit.Content)
		}
	}
	return respond
}

func buildSearchResponseButton(total, from int64, q string) tgbotapi.InlineKeyboardMarkup {
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

	idx := b.getIndex(callbackQuery.Message)
	res, err := idx.Search(string(q), &meilisearch.SearchRequest{
		Offset: from,
		Limit:  page,
	})
	if err != nil {
		reply = fmt.Sprintf("search error: %+v", err)
		return
	}

	msg := tgbotapi.NewEditMessageText(
		callbackQuery.Message.Chat.ID,
		callbackQuery.Message.MessageID,
		buildSearchResponse(b, callbackQuery.Message.Chat.ID, res, int(from)),
	)
	button := buildSearchResponseButton(res.NbHits, from, string(q))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &button

	_, err = b.Client.Send(msg)
	if err != nil {
		reply = fmt.Sprintf("update result error: %+v", err)
	} else {
		reply = "OK"
	}
}
