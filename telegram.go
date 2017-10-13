package main

import (
	"encoding/json"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	bt "github.com/ikool-cn/gobeanstalk-connection-pool"
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
	Tube        string
}

// NewTelegramBot ...
func NewTelegramBot(cfg *TelegramConfig, btdAddr string) (t *TelegramBot) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		logger.Panic("tg bot init failed:", err)
	}
	delay, err := time.ParseDuration(cfg.DeleteDelay)
	if err != nil {
		logger.Panic("delete delay error:", err)
	}

	t = &TelegramBot{
		Name:        bot.Self.UserName,
		SelfChatID:  cfg.SelfChatID,
		ComicPath:   cfg.ComicPath,
		DeleteDelay: delay,
		Client:      bot,
		Tube:        "tg",
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
	conn.Use(t.Tube)
	_, err = conn.Put(msg, 1, t.DeleteDelay, time.Minute)
	if err != nil {
		logger.Error(err)
		return
	}
}

func (t *TelegramBot) sendFile(chat int64, file string) (tgbotapi.Message, error) {
	logger.Debugf("[%d]%s", chat, file)
	if strings.HasSuffix(file, ".mp4") {
		return t.Client.Send(tgbotapi.NewVideoUpload(chat, file))
	}
	return t.Client.Send(tgbotapi.NewPhotoUpload(chat, file))

}

func (t *TelegramBot) delMessage() {
	for {
		conn, err := t.Queue.Get()
		if err != nil {
			logger.Error(err)
			time.Sleep(3 * time.Second)
			continue
		}
		conn.Watch(t.Tube)
		job, err := conn.Reserve()
		if err != nil {
			logger.Warning(err)
			time.Sleep(3 * time.Second)
			continue
		}
		msg := &tgbotapi.Message{}
		err = json.Unmarshal(job.Body, msg)
		if err != nil {
			logger.Error(err)
			err = conn.Bury(job.ID, 0)
			if err != nil {
				logger.Error(err)
			}
			time.Sleep(3 * time.Second)
			continue
		}
		delMsg := tgbotapi.DeleteMessageConfig{
			ChatID:    msg.Chat.ID,
			MessageID: msg.MessageID,
		}
		logger.Infof(":[%s]{%s}", getMsgTarget(msg), strconv.Quote(msg.Text))

		_, err = t.Client.DeleteMessage(delMsg)
		if err != nil {
			logger.Error(err)
			err = conn.Bury(job.ID, 0)
			if err != nil {
				logger.Error(err)
			}
			time.Sleep(3 * time.Second)
			continue
		}
		err = conn.Delete(job.ID)
		if err != nil {
			logger.Error(err)
			time.Sleep(3 * time.Second)
		}
		t.Queue.Release(conn, false)
	}
}

func (t *TelegramBot) tgBot() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	for {
		updates, err := t.Client.GetUpdatesChan(u)
		if err != nil {
			logger.Error(err)
			time.Sleep(3 * time.Second)
			continue
		}

		for update := range updates {
			if (update.Message == nil) && (update.EditedMessage == nil) {
				continue
			}
			if update.Message.Chat.IsGroup() {
				logger.Infof(
					"recv:(%s)[%s]{%s}",
					update.Message.Chat.Title,
					update.Message.From.String(),
					strconv.Quote(update.Message.Text))
			} else {
				logger.Infof(
					"recv:[%s]{%s}",
					update.Message.From.String(),
					strconv.Quote(update.Message.Text),
				)
			}

			if update.Message.IsCommand() {
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
		logger.Warning("tg bot restarted.")
		time.Sleep(3 * time.Second)
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
	files, err := filepath.Glob(twitterBot.ImgPath + "/*")
	if err != nil {
		logger.Error(err)
		return
	}
	if files == nil {
		logger.Error("find no pics")
	}
	rand.Seed(time.Now().Unix())
	file := files[rand.Intn(len(files))]

	logger.Infof("send:[%s]{%s}", getMsgTarget(update.Message), strconv.Quote(file))
	message, err := t.sendFile(update.Message.Chat.ID, file)
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

func getMsgTarget(m *tgbotapi.Message) string {
	if m.Chat.IsGroup() {
		return m.Chat.Title
	}
	if m.Chat.UserName != "" {
		return m.Chat.UserName
	}
	return m.Chat.FirstName + m.Chat.LastName
}
