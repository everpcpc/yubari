package main

import (
	"gopkg.in/telegram-bot-api.v4"
)

var (
	telegramBot *TelegramBot
)

// TelegramBot ...
type TelegramBot struct {
	Name       string
	SelfChatID int64
	Client     *tgbotapi.BotAPI
}

// NewTelegramBot ...
func NewTelegramBot(cfg *TelegramConfig) (t *TelegramBot) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		logger.Panic("tg bot init failed:", err)
	}
	t = &TelegramBot{
		Name:       bot.Self.UserName,
		SelfChatID: cfg.SelfChatID,
		Client:     bot,
	}
	return
}

func (t *TelegramBot) sendVideo(chat int64, file string) {
	logger.Infof("[%d]%s", chat, file)
	msg := tgbotapi.NewVideoUpload(chat, file)
	_, err := t.Client.Send(msg)
	if err != nil {
		logger.Error(err)
	}
}
func (t *TelegramBot) sendPhoto(chat int64, file string) {
	logger.Infof("[%d]%s", chat, file)
	msg := tgbotapi.NewPhotoUpload(chat, file)
	_, err := t.Client.Send(msg)
	if err != nil {
		logger.Error(err)
	}
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

		logger.Infof("[%s]{%d} %s", update.Message.From.UserName, update.Message.Chat.ID, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID

		t.Client.Send(msg)
	}
}
