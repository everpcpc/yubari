package main

import (
	"encoding/json"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	bt "github.com/ikool-cn/gobeanstalk-connection-pool"

	"gopkg.in/telegram-bot-api.v4"
)

var (
	telegramBot *TelegramBot
)

// TelegramBot ...
type TelegramBot struct {
	Name        string
	SelfChatID  int64
	ComicPath   string
	DeleteDelay time.Duration
	Client      *tgbotapi.BotAPI
	Queue       *bt.Pool
}

// NewTelegramBot ...
func NewTelegramBot(cfg *TelegramConfig, btdAddr string) (t *TelegramBot) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		logger.Panic("tg bot init failed:", err)
	}
	t = &TelegramBot{
		Name:        bot.Self.UserName,
		SelfChatID:  cfg.SelfChatID,
		ComicPath:   cfg.ComicPath,
		DeleteDelay: time.Duration(cfg.DeleteDelay),
		Client:      bot,
	}
	t.Queue = &bt.Pool{
		Dial: func() (*bt.Conn, error) {
			return bt.Dial(btdAddr)
		},
		MaxIdle:     10,
		MaxActive:   100,
		IdleTimeout: 60 * time.Second,
		MaxLifetime: 180 * time.Second,
		Wait:        true,
	}
	return
}

func (t *TelegramBot) putQueue(msg []byte) {
	conn, err := t.Queue.Get()
	if err != nil {
		logger.Error(err, msg)
		return
	}
	conn.Use("tg")
	_, err = conn.Put(msg, 1, t.DeleteDelay, time.Minute)
	if err != nil {
		logger.Error(err)
		return
	}
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
		case "start":
			go onStart(t, &update)
		case "comic":
			go onComic(t, &update)
		case "pic":
			go onPic(t, &update)
		default:
			logger.Info("ignore unkown cmd:", update.Message.Command())
			continue

		}

	}
}

func onStart(t *TelegramBot, update *tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "呀呀呀")
	msg.ReplyToMessageID = update.Message.MessageID
	t.Client.Send(msg)
}

func onComic(t *TelegramBot, update *tgbotapi.Update) {
	files, err := filepath.Glob(t.ComicPath)
	if err != nil {
		logger.Error(err)
		return
	}
	rand.Seed(time.Now().Unix())
	file := files[rand.Intn(len(files))]
	number := strings.Split(strings.Split(file, "@")[1], ".")[0]
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "https://nhentai.net/g/"+number)
	msg.ReplyToMessageID = update.Message.MessageID

	message, err := t.Client.Send(msg)
	if err != nil {
		logger.Error(err)
		return
	}
	data, err := json.Marshal(message)
	if err != nil {
		logger.Error(err)
		return
	}
	t.putQueue(data)
}

func onPic(t *TelegramBot, update *tgbotapi.Update) {
	_, err := filepath.Glob(twitterBot.ImgPath + "/")
	if err != nil {
		logger.Error(err)
		return
	}
}
