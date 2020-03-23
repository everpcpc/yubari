package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

func onSearch(b *Bot, message *tgbotapi.Message) {
	args := message.CommandArguments()
	msg := tgbotapi.NewMessage(message.Chat.ID, "呀呀呀"+args)

	msg.ReplyToMessageID = message.MessageID
	b.Client.Send(msg)
}
