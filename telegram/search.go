package telegram

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/everpcpc/yubari/elasticsearch"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
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
	res, err := elasticsearch.SearchMessage(b.es, idx, q, 0)
	if err != nil {
		b.logger.Errorf("es search error: %+v", err)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, buildSearchResponse(res, 0))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = buildSearchResponseButton(res, 0)

	msgSent, err := b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
	if b.redis.Set(getQueryCacheKey(&msgSent), q, 0).Err() != nil {
		b.logger.Errorf("cache query failed: %+v", err)
	}
}

func getQueryCacheKey(msg *tgbotapi.Message) string {
	return fmt.Sprintf("search_query_%d", msg.MessageID)
}

func getIndex(message *tgbotapi.Message) string {
	return fmt.Sprintf("%s-%d", message.Chat.Type, message.Chat.ID)
}

func buildSearchResponse(res *elasticsearch.SearchResponse, from int) string {
	total := res.Hits.Total.Value
	respond := fmt.Sprintf("搜素到 %d 个结果：\n", total)
	for i, hit := range res.Hits.Hits {
		var content string
		if len(hit.Highlight.Content) == 0 {
			content = hit.Source.Content[:15]
		} else {
			content = hit.Highlight.Content[0]
		}
		respond += fmt.Sprintf("%d. <a href=\"%d\">%s</a>\n", from+i+1, hit.Source.MessageID, content)
	}
	respond += fmt.Sprintf("耗时 %.3f 秒。", float64(res.Took)/1000)
	return respond
}

func buildSearchResponseButton(res *elasticsearch.SearchResponse, from int) tgbotapi.InlineKeyboardMarkup {
	total := res.Hits.Total.Value
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ 上一页", fmt.Sprintf("search:%d", max(uint64(from-5), 0))),
		tgbotapi.NewInlineKeyboardButtonData("下一页 ➡️", fmt.Sprintf("search:%d", min(uint64(from+5), uint64(total)))),
	)
	return tgbotapi.NewInlineKeyboardMarkup(row)
}

func onReactionSearch(b *Bot, callbackQuery *tgbotapi.CallbackQuery) {
	token := strings.Split(callbackQuery.Data, ":")
	if len(token) != 2 {
		b.logger.Errorf("react data error: %s", callbackQuery.Data)
		return
	}

	from, err := strconv.ParseInt(token[1], 10, 0)
	if err != nil {
		b.logger.Errorf("react search from error: %s", callbackQuery.Data)
	}

	q := b.redis.Get(getQueryCacheKey(callbackQuery.Message)).String()
	if q == "" {
		b.logger.Warningf("query not found for reaction: %s", callbackQuery.Data)
	}

	idx := getIndex(callbackQuery.Message)

	res, err := elasticsearch.SearchMessage(b.es, idx, q, int(from))
	if err != nil {
		b.logger.Errorf("es search error: %+v", err)
		return
	}

	msg := tgbotapi.NewEditMessageText(
		callbackQuery.Message.Chat.ID,
		callbackQuery.Message.MessageID,
		buildSearchResponse(res, int(from)),
	)
	button := buildSearchResponseButton(res, int(from))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &button

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("search reaction reply error: %+v", err)
	}
}
