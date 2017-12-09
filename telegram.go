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
		logger.Panicf("tg bot init failed: %+v", err)
	}
	delay, err := time.ParseDuration(cfg.DeleteDelay)
	if err != nil {
		logger.Panicf("delete delay error: %+v", err)
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
		logger.Errorf("%+v: %s", err, string(msg))
		return
	}
	conn.Use(t.Tube)
	_, err = conn.Put(msg, 1, t.DeleteDelay, time.Minute)
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}
}

func (t *TelegramBot) sendFile(chat int64, file string) (tgbotapi.Message, error) {
	logger.Debugf("[%d]%s", chat, file)
	return t.Client.Send(tgbotapi.NewDocumentUpload(chat, file))
	// if strings.HasSuffix(file, ".mp4") {
	// return t.Client.Send(tgbotapi.NewVideoUpload(chat, file))
	// }
	// return t.Client.Send(tgbotapi.NewPhotoUpload(chat, file))

}

func (t *TelegramBot) delMessage() {
	for {
		conn, err := t.Queue.Get()
		if err != nil {
			logger.Errorf("%+v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		conn.Watch(t.Tube)
		job, err := conn.Reserve()
		if err != nil {
			logger.Warningf("%+v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		msg := &tgbotapi.Message{}
		err = json.Unmarshal(job.Body, msg)
		if err != nil {
			logger.Errorf("%+v", err)
			err = conn.Bury(job.ID, 0)
			if err != nil {
				logger.Errorf("%+v", err)
			}
			time.Sleep(3 * time.Second)
			continue
		}
		delMsg := tgbotapi.DeleteMessageConfig{
			ChatID:    msg.Chat.ID,
			MessageID: msg.MessageID,
		}
		logger.Infof(":[%s]{%s}", getMsgTitle(msg), strconv.Quote(msg.Text))

		_, err = t.Client.DeleteMessage(delMsg)
		if err != nil {
			logger.Errorf("%+v", err)
			err = conn.Bury(job.ID, 0)
			if err != nil {
				logger.Errorf("%+v", err)
			}
			time.Sleep(3 * time.Second)
			continue
		}
		err = conn.Delete(job.ID)
		if err != nil {
			logger.Errorf("%+v", err)
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
			logger.Errorf("%+v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		var message *tgbotapi.Message
		for update := range updates {
			if update.Message != nil {
				message = update.Message
			} else if update.EditedMessage != nil {
				message = update.EditedMessage
			} else {
				// unkown msg type
				continue
			}
			if message.Chat.IsGroup() {
				logger.Infof(
					"recv:(%s)[%s]{%s}",
					message.Chat.Title,
					message.From.String(),
					strconv.Quote(message.Text))
			} else {
				logger.Infof(
					"recv:[%s]{%s}",
					message.From.String(),
					strconv.Quote(message.Text),
				)
			}

			if message.IsCommand() {
				switch message.Command() {
				case "start":
					go onStart(t, message)
				case "comic":
					go onComic(t, message)
				case "pic":
					go onPic(t, message)
				default:
					logger.Infof("ignore unkown cmd: %+v", message.Command())
					continue
				}
			} else {
				if message.Text == "" {
					continue
				}
				checkRepeat(t, message)
			}
		}
		logger.Warning("tg bot restarted.")
		time.Sleep(3 * time.Second)
	}
}

func checkRepeat(t *TelegramBot, message *tgbotapi.Message) {
	key := "tg_" + getMsgTitle(message) + "_last"
	flattendMsg := strings.TrimSpace(message.Text)
	defer redisClient.LTrim(key, 0, 10)
	defer redisClient.LPush(key, flattendMsg)

	lastMsgs, err := redisClient.LRange(key, 0, 6).Result()
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}
	i := 0
	for _, s := range lastMsgs {
		if s == flattendMsg {
			i++
		}
	}
	if i > 1 {
		redisClient.Del(key)
		logger.Infof("repeat: %s", strconv.Quote(message.Text))
		msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)
		t.Client.Send(msg)
	}
}

func onStart(t *TelegramBot, message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "呀呀呀")
	msg.ReplyToMessageID = message.MessageID
	t.Client.Send(msg)
}

func onComic(t *TelegramBot, message *tgbotapi.Message) {
	files, err := filepath.Glob(t.ComicPath)
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}
	rand.Seed(time.Now().Unix())
	file := files[rand.Intn(len(files))]
	number := strings.Split(strings.Split(file, "@")[1], ".")[0]
	msg := tgbotapi.NewMessage(message.Chat.ID, "https://nhentai.net/g/"+number)

	logger.Infof("send:[%s]{%s}", getMsgTitle(message), strconv.Quote(file))
	msgSent, err := t.Client.Send(msg)
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}
	data, err := json.Marshal(msgSent)
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}
	t.putQueue(data)
}

func onPic(t *TelegramBot, message *tgbotapi.Message) {
	files, err := filepath.Glob(twitterBot.ImgPath + "/*")
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}
	if files == nil {
		logger.Error("find no pics")
	}
	rand.Seed(time.Now().Unix())
	file := files[rand.Intn(len(files))]

	logger.Infof("send:[%s]{%s}", getMsgTitle(message), strconv.Quote(file))
	msgSent, err := t.sendFile(message.Chat.ID, file)
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}
	data, err := json.Marshal(msgSent)
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}
	t.putQueue(data)
}

func getMsgTitle(m *tgbotapi.Message) string {
	if m.Chat.IsGroup() {
		return m.Chat.Title
	}
	return m.From.String()
}
