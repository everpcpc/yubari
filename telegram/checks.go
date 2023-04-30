package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/net/html"

	"yubari/meili"
	"yubari/pixiv"
)

func checkRepeat(b *Bot, message *tgbotapi.Message) {

	key := "tg_last_" + strconv.FormatInt(message.Chat.ID, 10)
	flattendMsg := strings.TrimSpace(message.Text)
	defer b.redis.LTrim(key, 0, 10)
	defer b.redis.LPush(key, flattendMsg)

	lastMsgs, err := b.redis.LRange(key, 0, 6).Result()
	if err != nil {
		b.logger.Errorf("%s", err)
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
	sizes, err := b.pixivBot.Download(id)
	if err != nil {
		callbackText += fmt.Sprintf("😕 download error: %s", err)
	} else {
		for i := range sizes {
			if sizes[i] == 0 {
				callbackText += fmt.Sprintf("p%d: exists😋 ", i)
				continue
			}
			b.logger.Debugf("download pixiv %d_p%d: %s", id, i, ByteCountIEC(sizes[i]))
			callbackText += fmt.Sprintf("p%d: %s😊 ", i, byteCountBinary(sizes[i]))
		}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, callbackText)
	msg.ReplyToMessageID = message.MessageID

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%s", err)
	}
}

func checkSave(b *Bot, message *tgbotapi.Message) {
	idx := b.getIndex(message)

	article := meili.Article{
		ID:      int64(message.MessageID),
		User:    int64(message.From.ID),
		Date:    int64(message.Date),
		Content: html.EscapeString(message.Text),
	}
	_, err := idx.AddDocuments(&article, "id")
	if err != nil {
		b.logger.Errorf("save message error: %s", err)
	}
}

func checkOpenAI(b *Bot, message *tgbotapi.Message) {
	if !b.isAuthedChat(message.Chat) {
		return
	}

	enabled := false
	if message.Chat.IsPrivate() {
		enabled = true
	} else if strings.HasPrefix(message.Text, "@yubari_bot") {
		enabled = true
	}
	submessage := message.ReplyToMessage
	if submessage != nil {
		if submessage.From.ID == b.Client.Self.ID {
			enabled = true
		}
	}

	if !enabled {
		return
	}

	b.setChatAction(message.Chat.ID, "typing")

	m := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: message.Text,
	}
	chatMessages := []openai.ChatCompletionMessage{m}
	for submessage != nil {
		m := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: submessage.Text,
		}
		if submessage.From.ID == b.Client.Self.ID {
			m.Role = openai.ChatMessageRoleAssistant
		}
		chatMessages = append([]openai.ChatCompletionMessage{m}, chatMessages...)
		submessage = submessage.ReplyToMessage
	}
	resp, err := b.ai.CreateChatCompletion(context.TODO(), openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		Messages:    chatMessages,
		Temperature: 0.0,
	})

	content := ""
	if err != nil {
		content = "😕 openai error, please try again"
		b.logger.Errorf("openai request error: %s", err)
	} else {
		content = resp.Choices[0].Message.Content
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, content)
	msg.ReplyToMessageID = message.MessageID

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("openai reply error: %s", err)
	}
}
