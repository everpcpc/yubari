package main

import (
	"strconv"

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

func (t *TelegramBot) sendFile(chat int64, file string, mediaType string) {
	logger.Infof("[%d]%s", chat, file)
	var err error
	switch mediaType {
	case "photo":
		_, err = t.Client.Send(tgbotapi.NewPhotoUpload(chat, file))
	case "video":
		_, err = t.Client.Send(tgbotapi.NewVideoUpload(chat, file))
	case "animated_gif":
		_, err = t.Client.Send(tgbotapi.NewVideoUpload(chat, file))
	default:
		logger.Notice("media type ignored:", mediaType)
	}
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
		if !update.Message.IsCommand() {
			continue
		}

		if update.Message.Chat.IsGroup() {
			logger.Infof(
				"[%s](%s){%s}",
				update.Message.From.UserName,
				update.Message.Chat.Title,
				strconv.Quote(update.Message.Text))
		} else {
			logger.Infof("[%s]{%s}", update.Message.From.UserName, strconv.Quote(update.Message.Text))
		}

		switch update.Message.Command() {
		case "test":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "呀呀呀")
			msg.ReplyToMessageID = update.Message.MessageID
			t.Client.Send(msg)
		default:

		}

	}
}
