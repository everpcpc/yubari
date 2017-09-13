package main

import (
	"gopkg.in/telegram-bot-api.v4"
)

var (
	telegramBot *TelegramBot
)

// TelegramBot ...
type TelegramBot struct {
	Name   string
	Client *tgbotapi.BotAPI
}

// NewTelegramBot ...
func NewTelegramBot(cfg *TelegramConfig) (t *TelegramBot) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		logger.Panic("tg bot init failed:", err)
	}
	t = &TelegramBot{
		Name:   bot.Self.UserName,
		Client: bot,
	}
	return
}

func (t *TelegramBot) tgBot() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := t.Client.GetUpdatesChan(u)
	if err != nil {
		logger.Error(err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		logger.Infof("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID

		t.Client.Send(msg)
	}
}
