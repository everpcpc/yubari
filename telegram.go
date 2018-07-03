package main

import (
	"encoding/json"
	"fmt"
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
	Name          string
	SelfChatID    int64
	ChannelChatID int64
	ComicPath     string
	DeleteDelay   time.Duration
	Client        *tgbotapi.BotAPI
	Queue         *bt.Pool
	Tube          string
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
		Name:          bot.Self.UserName,
		SelfChatID:    cfg.SelfChatID,
		ChannelChatID: cfg.ChannelChatID,
		ComicPath:     cfg.ComicPath,
		DeleteDelay:   delay,
		Client:        bot,
		Tube:          "tg",
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

func (t *TelegramBot) send(chat int64, msg string) (tgbotapi.Message, error) {
	logger.Debugf("[%d]%s", chat, msg)
	return t.Client.Send(tgbotapi.NewMessage(chat, msg))
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
			} else if update.CallbackQuery != nil {
				logger.Infof(
					"recv:(%s)[%s]reaction:{%s}",
					update.CallbackQuery.ChatInstance,
					update.CallbackQuery.From.String(),
					update.CallbackQuery.Data,
				)
				data := strings.SplitN(update.CallbackQuery.Data, ":", 2)
				switch data[0] {
				case "comic", "pic":
					go onReaction(t, update.CallbackQuery)
				}
				continue
			} else {
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
	key := "tg_last_" + strconv.FormatInt(message.Chat.ID, 10)
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
	rand.Seed(time.Now().UnixNano())
	file := files[rand.Intn(len(files))]
	number := strings.Split(strings.Split(file, "@")[1], ".")[0]
	msg := tgbotapi.NewMessage(message.Chat.ID, "🔞 https://nhentai.net/g/"+number)

	msg.ReplyMarkup = buildInlineKeyboardMarkup("comic", number)

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
	files, err := filepath.Glob(filepath.Join(twitterBot.ImgPath, "*"))
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}
	if files == nil {
		logger.Error("find no pics")
	}
	rand.Seed(time.Now().UnixNano())
	file := files[rand.Intn(len(files))]

	logger.Infof("send:[%s]{%s}", getMsgTitle(message), strconv.Quote(file))

	msg := tgbotapi.NewDocumentUpload(message.Chat.ID, file)
	msg.ReplyMarkup = buildInlineKeyboardMarkup("pic", filepath.Base(file))

	_, err = t.Client.Send(msg)
	if err != nil {
		logger.Errorf("%+v", err)
	}
}

func onReaction(t *TelegramBot, callbackQuery *tgbotapi.CallbackQuery) {
	var callbackText string

	_type, _id, reaction, diss, err := saveReaction(callbackQuery.Data, callbackQuery.From.ID)
	if err == nil {
		if diss <= 1 {
			msg := tgbotapi.NewEditMessageReplyMarkup(
				callbackQuery.Message.Chat.ID,
				callbackQuery.Message.MessageID,
				buildInlineKeyboardMarkup(_type, _id),
			)
			_, err = t.Client.Send(msg)
		} else {
			delMsg := tgbotapi.DeleteMessageConfig{
				ChatID:    callbackQuery.Message.Chat.ID,
				MessageID: callbackQuery.Message.MessageID,
			}
			_, err = t.Client.DeleteMessage(delMsg)
			if err == nil {
				err = probate(_type, _id)
			}
		}
	}

	if err != nil {
		logger.Debugf("%+v", err)
		callbackText = err.Error()
	} else {
		callbackText = reaction + " " + _id + "!"
	}

	callbackMsg := tgbotapi.NewCallback(callbackQuery.ID, callbackText)
	_, err = t.Client.AnswerCallbackQuery(callbackMsg)
	if err != nil {
		logger.Errorf("%+v", err)
	}
}

func getMsgTitle(m *tgbotapi.Message) string {
	if m.Chat.IsGroup() {
		return m.Chat.Title
	}
	return m.From.String()
}

func buildReactionData(_type, _id, reaction string) string {
	return _type + ":" + _id + ":" + reaction
}
func buildReactionKey(_type, _id, reaction string) string {
	return "reaction_" + buildReactionData(_type, _id, reaction)
}

func buildInlineKeyboardMarkup(_type, _id string) tgbotapi.InlineKeyboardMarkup {

	likeCount, _ := redisClient.SCard(buildReactionKey(_type, _id, "like")).Result()
	dissCount, _ := redisClient.SCard(buildReactionKey(_type, _id, "diss")).Result()

	likeText := "❤️"
	if likeCount > 0 {
		likeText = likeText + " " + strconv.FormatInt(likeCount, 10)
	}
	dissText := "💔"
	if dissCount > 0 {
		dissText = dissText + " " + strconv.FormatInt(dissCount, 10)
	}

	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(likeText, buildReactionData(_type, _id, "like")),
		tgbotapi.NewInlineKeyboardButtonData(dissText, buildReactionData(_type, _id, "diss")),
	)
	return tgbotapi.NewInlineKeyboardMarkup(row)
}

func saveReaction(key string, user int) (_type, _id, reaction string, diss int64, err error) {
	token := strings.Split(key, ":")
	if len(token) != 3 {
		err = fmt.Errorf("react data error: %s", key)
		return
	}
	_type = token[0]
	_id = token[1]
	reaction = token[2]

	pipe := redisClient.Pipeline()
	switch reaction {
	case "like":
		likeCount := pipe.SAdd(buildReactionKey(_type, _id, "like"), strconv.Itoa(user))
		dissCount := pipe.SRem(buildReactionKey(_type, _id, "diss"), strconv.Itoa(user))
		_, err = pipe.Exec()
		if err == nil {
			if likeCount.Val()+dissCount.Val() == 0 {
				err = fmt.Errorf("not modified")
			}
		}
	case "diss":
		dissCount := pipe.SAdd(buildReactionKey(_type, _id, "diss"), strconv.Itoa(user))
		likeCount := pipe.SRem(buildReactionKey(_type, _id, "like"), strconv.Itoa(user))
		_, err = pipe.Exec()
		if err == nil {
			if likeCount.Val()+dissCount.Val() == 0 {
				err = fmt.Errorf("not modified")
			}
		}
		diss = dissCount.Val()
	default:
		err = fmt.Errorf("react type error: %s", key)
	}
	return
}
