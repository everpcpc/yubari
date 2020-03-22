package telegram

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	bt "github.com/ikool-cn/gobeanstalk-connection-pool"
	logging "github.com/op/go-logging"

	"github.com/everpcpc/yubari/pixiv"
)

type Config struct {
	Token          string  `json:"token"`
	SelfID         int64   `json:"selfID"`
	WhitelistChats []int64 `json:"whitelistChats"`
	ComicPath      string  `json:"comicPath"`
	DeleteDelay    string  `json:"deleteDelay"`
}

type Bot struct {
	Name           string
	SelfID         int64
	WhitelistChats []int64
	ComicPath      string
	PixivPath      string
	TwitterImgPath string
	DeleteDelay    time.Duration
	Client         *tgbotapi.BotAPI
	Queue          *bt.Pool
	Tube           string
	logger         *logging.Logger
	redis          *redis.Client
}

func NewBot(cfg *Config) (b *Bot, err error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("tg bot init failed: %+v", err)
	}
	delay, err := time.ParseDuration(cfg.DeleteDelay)
	if err != nil {
		return nil, fmt.Errorf("delete delay error: %+v", err)
	}

	b = &Bot{
		Name:           bot.Self.UserName,
		SelfID:         cfg.SelfID,
		WhitelistChats: cfg.WhitelistChats,
		ComicPath:      cfg.ComicPath,
		DeleteDelay:    delay,
		Client:         bot,
	}
	return
}

func (b *Bot) WithLogger(logger *logging.Logger) *Bot {
	b.logger = logger
	return b
}

func (b *Bot) WithRedis(rds *redis.Client) *Bot {
	b.redis = rds
	return b
}

func (b *Bot) WithPixivImg(imgPath string) *Bot {
	b.PixivPath = imgPath
	return b
}

func (b *Bot) WithTwitterImg(imgPath string) *Bot {
	b.TwitterImgPath = imgPath
	return b
}

func (b *Bot) WithBeanstalkd(addr string) *Bot {
	b.Queue = &bt.Pool{
		Dial: func() (*bt.Conn, error) {
			return bt.Dial(addr)
		},
		MaxIdle:     10,
		MaxActive:   100,
		IdleTimeout: 60 * time.Second,
		MaxLifetime: 180 * time.Second,
		Wait:        true,
	}
	b.Tube = "tg"
	return b
}

func (b *Bot) putQueue(msg []byte) {
	conn, err := b.Queue.Get()
	if err != nil {
		b.logger.Errorf("%+v: %s", err, string(msg))
		return
	}
	conn.Use(b.Tube)
	_, err = conn.Put(msg, 1, b.DeleteDelay, time.Minute)
	if err != nil {
		b.logger.Errorf("%+v", err)
		return
	}
}

func (b *Bot) isAuthedChat(c *tgbotapi.Chat) bool {
	for _, w := range b.WhitelistChats {
		if c.ID == w {
			return true
		}
	}
	return false
}

func (b *Bot) Send(chat int64, msg string) (tgbotapi.Message, error) {
	b.logger.Debugf("[%d]%s", chat, msg)
	return b.Client.Send(tgbotapi.NewMessage(chat, msg))
}

func (b *Bot) SendPixivIllust(target int64, id uint64) {
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⭕️", buildReactionData("pixivIllust", strconv.FormatUint(id, 10), "like")),
		tgbotapi.NewInlineKeyboardButtonData("❌", buildReactionData("pixivIllust", strconv.FormatUint(id, 10), "diss")),
	)
	msg := tgbotapi.NewMessage(target, pixiv.URLWithID(id))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(row)
	_, err := b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func (b *Bot) startDeleteMessage() {
	for {
		conn, err := b.Queue.Get()
		if err != nil {
			b.logger.Errorf("%+v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		conn.Watch(b.Tube)
		job, err := conn.Reserve()
		if err != nil {
			b.logger.Warningf("%+v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		msg := &tgbotapi.Message{}
		err = json.Unmarshal(job.Body, msg)
		if err != nil {
			b.logger.Errorf("%+v", err)
			err = conn.Bury(job.ID, 0)
			if err != nil {
				b.logger.Errorf("%+v", err)
			}
			time.Sleep(3 * time.Second)
			continue
		}
		delMsg := tgbotapi.DeleteMessageConfig{
			ChatID:    msg.Chat.ID,
			MessageID: msg.MessageID,
		}
		b.logger.Infof(":[%s]{%s}", getMsgTitle(msg), strconv.Quote(msg.Text))

		_, err = b.Client.DeleteMessage(delMsg)
		if err != nil {
			b.logger.Errorf("%+v", err)
			err = conn.Bury(job.ID, 0)
			if err != nil {
				b.logger.Errorf("%+v", err)
			}
			time.Sleep(3 * time.Second)
			continue
		}
		err = conn.Delete(job.ID)
		if err != nil {
			b.logger.Errorf("%+v", err)
			time.Sleep(3 * time.Second)
		}
		b.Queue.Release(conn, false)
	}
}

func (b *Bot) Start() {
	go b.startDeleteMessage()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	for {
		updates, err := b.Client.GetUpdatesChan(u)
		if err != nil {
			b.logger.Errorf("%+v", err)
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
				b.logger.Infof(
					"recv:(%d)[%s]reaction:{%s}",
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.From.String(),
					update.CallbackQuery.Data,
				)
				data := strings.SplitN(update.CallbackQuery.Data, ":", 2)
				switch data[0] {
				case "comic", "pic", "pixiv":
					go onReaction(b, update.CallbackQuery)
				case "pixivIllust":
					if !b.isAuthedChat(update.CallbackQuery.Message.Chat) {
						b.logger.Warning("reaction from illegal chat, ignore")
						break
					}
					go onReactionSelf(b, update.CallbackQuery)
				default:
				}
				continue
			} else {
				continue
			}
			if message.Chat.IsGroup() {
				b.logger.Infof(
					"recv:(%d)[%s:%s]{%s}",
					message.Chat.ID,
					message.Chat.Title,
					message.From.String(),
					strconv.Quote(message.Text))
			} else {
				b.logger.Infof(
					"recv:(%d)[%s]{%s}",
					message.Chat.ID,
					message.From.String(),
					strconv.Quote(message.Text),
				)
			}

			if message.IsCommand() {
				switch message.Command() {
				case "start":
					go onStart(b, message)
				case "roll":
					go onRoll(b, message)
				case "comic":
					go onComic(b, message)
				case "pic":
					go onPic(b, message)
				case "pixiv":
					go onPixiv(b, message)
				default:
					b.logger.Infof("ignore unknown cmd: %+v", message.Command())
					continue
				}
			} else {
				if message.Text == "" {
					continue
				}
				checkRepeat(b, message)
				checkPixiv(b, message)
			}
		}
		b.logger.Warning("tg bot restarted.")
		time.Sleep(3 * time.Second)
	}
}

func checkRepeat(b *Bot, message *tgbotapi.Message) {
	key := "tg_last_" + strconv.FormatInt(message.Chat.ID, 10)
	flattendMsg := strings.TrimSpace(message.Text)
	defer b.redis.LTrim(key, 0, 10)
	defer b.redis.LPush(key, flattendMsg)

	lastMsgs, err := b.redis.LRange(key, 0, 6).Result()
	if err != nil {
		b.logger.Errorf("%+v", err)
		return
	}
	i := 0
	for _, s := range lastMsgs {
		if s == flattendMsg {
			i++
		}
	}
	if i > 1 {
		b.redis.Del(key)
		b.logger.Infof("repeat: %s", strconv.Quote(message.Text))
		msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)
		go b.Client.Send(msg)
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
	var callbackText string
	sizes, errs := pixiv.Download(id, b.PixivPath)
	for i := range sizes {
		if errs[i] != nil {
			callbackText += fmt.Sprintf("p%d: error😕 ", i)
			continue
		}
		if sizes[i] == 0 {
			callbackText += fmt.Sprintf("p%d: exists😋 ", i)
			continue
		}
		b.logger.Debugf("download pixiv %d_p%d: %d bytes", id, i, sizes[i])
		callbackText += fmt.Sprintf("p%d: %s😊 ", i, byteCountBinary(sizes[i]))
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, callbackText)
	msg.ReplyToMessageID = message.MessageID

	_, err := b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func onStart(b *Bot, message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "呀呀呀")
	msg.ReplyToMessageID = message.MessageID
	b.Client.Send(msg)
}

func onRoll(b *Bot, message *tgbotapi.Message) {
	var err error
	var limit int

	args := message.CommandArguments()

	if args != "" {
		limit, err = strconv.Atoi(args)
		if err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "输入不对啦")
			msg.ReplyToMessageID = message.MessageID
			_, err := b.Client.Send(msg)
			if err != nil {
				b.logger.Errorf("%+v", err)
			}
			return
		}
	}
	if limit <= 0 {
		limit = 100
	}

	rand.Seed(time.Now().UnixNano())
	msg := tgbotapi.NewMessage(message.Chat.ID, "🎲 "+strconv.Itoa(rand.Intn(limit)))
	msg.ReplyToMessageID = message.MessageID
	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func onComic(b *Bot, message *tgbotapi.Message) {
	files, err := filepath.Glob(filepath.Join(b.ComicPath, "*.epub"))
	if err != nil {
		b.logger.Errorf("%+v", err)
		return
	}
	if files == nil {
		b.logger.Error("find no comic")
		return
	}
	rand.Seed(time.Now().UnixNano())
	file := files[rand.Intn(len(files))]
	number := strings.Split(strings.Split(file, "@")[1], ".")[0]
	msg := tgbotapi.NewMessage(message.Chat.ID, "🔞 https://nhentai.net/g/"+number)

	msg.ReplyMarkup = b.buildInlineKeyboardMarkup("comic", number)

	b.logger.Infof("send:[%s]{%s}", getMsgTitle(message), strconv.Quote(file))
	msgSent, err := b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
		return
	}
	data, err := json.Marshal(msgSent)
	if err != nil {
		b.logger.Errorf("%+v", err)
		return
	}
	b.putQueue(data)
}

func onPic(b *Bot, message *tgbotapi.Message) {
	files, err := filepath.Glob(filepath.Join(b.TwitterImgPath, "*"))
	if err != nil {
		b.logger.Errorf("%+v", err)
		return
	}
	if files == nil {
		b.logger.Error("find no pic")
		return
	}
	rand.Seed(time.Now().UnixNano())
	file := files[rand.Intn(len(files))]

	b.logger.Infof("send:[%s]{%s}", getMsgTitle(message), strconv.Quote(file))

	msg := tgbotapi.NewDocumentUpload(message.Chat.ID, file)
	msg.ReplyMarkup = b.buildInlineKeyboardMarkup("pic", filepath.Base(file))

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func onPixiv(b *Bot, message *tgbotapi.Message) {
	args := message.CommandArguments()

	if args != "" {
		if id, err := strconv.ParseUint(args, 10, 0); err == nil {
			b.SendPixivIllust(message.Chat.ID, id)
			return
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, "输入不对啦")
		msg.ReplyToMessageID = message.MessageID
		msgSent, err := b.Client.Send(msg)
		if err != nil {
			b.logger.Errorf("%+v", err)
			return
		}
		data, err := json.Marshal(msgSent)
		if err != nil {
			b.logger.Errorf("%+v", err)
			return
		}
		b.putQueue(data)
		return
	}
	files, err := filepath.Glob(filepath.Join(b.PixivPath, "*"))
	if err != nil {
		b.logger.Errorf("%+v", err)
		return
	}
	if files == nil {
		b.logger.Error("find no pic")
		return
	}
	rand.Seed(time.Now().UnixNano())
	file := files[rand.Intn(len(files))]
	b.logger.Infof("send:[%s]{%s}", getMsgTitle(message), strconv.Quote(file))
	msg := tgbotapi.NewDocumentUpload(message.Chat.ID, file)
	msg.ReplyMarkup = b.buildInlineKeyboardMarkup("pixiv", filepath.Base(file))
	msg.ReplyToMessageID = message.MessageID

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func onReaction(b *Bot, callbackQuery *tgbotapi.CallbackQuery) {
	var callbackText string

	_type, _id, reaction, err := b.saveReaction(callbackQuery.Data, callbackQuery.From.ID)
	if err == nil {
		diss := b.redis.SCard(buildReactionKey(_type, _id, "diss")).Val()
		like := b.redis.SCard(buildReactionKey(_type, _id, "like")).Val()
		if diss-like < 2 {
			msg := tgbotapi.NewEditMessageReplyMarkup(
				callbackQuery.Message.Chat.ID,
				callbackQuery.Message.MessageID,
				b.buildInlineKeyboardMarkup(_type, _id),
			)
			_, err = b.Client.Send(msg)
		} else {
			delMsg := tgbotapi.DeleteMessageConfig{
				ChatID:    callbackQuery.Message.Chat.ID,
				MessageID: callbackQuery.Message.MessageID,
			}
			_, err = b.Client.DeleteMessage(delMsg)
			if err == nil {
				err = b.probate(_type, _id)
			}
		}
	}

	if err != nil {
		b.logger.Debugf("%+v", err)
		callbackText = err.Error()
	} else {
		callbackText = reaction + " " + _id + "!"
	}

	callbackMsg := tgbotapi.NewCallback(callbackQuery.ID, callbackText)
	_, err = b.Client.AnswerCallbackQuery(callbackMsg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func onReactionSelf(b *Bot, callbackQuery *tgbotapi.CallbackQuery) {

	var callbackText string

	token := strings.Split(callbackQuery.Data, ":")
	if len(token) != 3 {
		b.logger.Errorf("react data error: %s", callbackQuery.Data)
		return
	}
	_id := token[1]
	reaction := token[2]
	switch reaction {
	case "like":
		id, err := strconv.ParseUint(_id, 10, 0)
		if err != nil {
			callbackText = "failed parsing pixiv id"
			break
		}
		sizes, errs := pixiv.Download(id, b.PixivPath)
		for i := range sizes {
			if errs[i] != nil {
				callbackText += fmt.Sprintf("p%d: error;", i)
				continue
			}
			if sizes[i] == 0 {
				callbackText += fmt.Sprintf("p%d: exists;", i)
				continue
			}
			b.logger.Debugf("download pixiv %d_p%d: %d bytes", id, i, sizes[i])
			callbackText += fmt.Sprintf("p%d: %s;", i, byteCountBinary(sizes[i]))
		}

	case "diss":
	default:
		callbackText = fmt.Sprintf("react type error: %s", reaction)
	}

	delMsg := tgbotapi.DeleteMessageConfig{
		ChatID:    callbackQuery.Message.Chat.ID,
		MessageID: callbackQuery.Message.MessageID,
	}
	_, err := b.Client.DeleteMessage(delMsg)
	if err != nil {
		b.logger.Errorf("failed deleting msg: %+v", err)
	}

	callbackMsg := tgbotapi.NewCallback(callbackQuery.ID, callbackText)
	_, err = b.Client.AnswerCallbackQuery(callbackMsg)
	if err != nil {
		b.logger.Errorf("%+v", err)
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

func (b *Bot) buildInlineKeyboardMarkup(_type, _id string) tgbotapi.InlineKeyboardMarkup {

	likeCount, _ := b.redis.SCard(buildReactionKey(_type, _id, "like")).Result()
	dissCount, _ := b.redis.SCard(buildReactionKey(_type, _id, "diss")).Result()

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

func (b *Bot) saveReaction(key string, user int) (_type, _id, reaction string, err error) {
	token := strings.Split(key, ":")
	if len(token) != 3 {
		err = fmt.Errorf("react data error: %s", key)
		return
	}
	_type = token[0]
	_id = token[1]
	reaction = token[2]

	pipe := b.redis.Pipeline()
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
	default:
		err = fmt.Errorf("react type error: %s", key)
	}
	return
}

func (b *Bot) probate(_type, _id string) error {
	b.logger.Noticef("%s: %s", _type, _id)
	switch _type {
	case "comic":
		fileName := "nhentai.net@" + _id + ".epub"
		return os.Rename(
			filepath.Join(b.ComicPath, fileName),
			filepath.Join(b.ComicPath, "probation", fileName),
		)
	case "pic":
		return os.Rename(
			filepath.Join(b.TwitterImgPath, _id),
			filepath.Join(b.TwitterImgPath, "probation", _id),
		)
	case "pixiv":
		return os.Rename(
			filepath.Join(b.PixivPath, _id),
			filepath.Join(b.PixivPath, "probation", _id),
		)
	default:
		return fmt.Errorf("prohibit unkown type")
	}
}
