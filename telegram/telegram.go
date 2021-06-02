package telegram

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	bt "github.com/ikool-cn/gobeanstalk-connection-pool"
	logging "github.com/op/go-logging"

	"yubari/pixiv"
)

const (
	tgDeleteTube = "tg_delete"
	tgPixivTube  = "tg_pixiv"
)

type Config struct {
	Token          string  `json:"token"`
	SelfID         int64   `json:"selfID"`
	WhitelistChats []int64 `json:"whitelistChats"`
	ComicPath      string  `json:"comicPath"`
	DeleteDelay    string  `json:"deleteDelay"`
}

type DownloadPixiv struct {
	ChatID    int64
	MessageID int
	PixivID   uint64
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
	logger         *logging.Logger
	redis          *redis.Client
	es             *elasticsearch7.Client
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

func (b *Bot) WithQueue(queue *bt.Pool) *Bot {
	b.Queue = queue
	return b
}

func (b *Bot) WithES(es *elasticsearch7.Client) *Bot {
	b.es = es
	return b
}

func (b *Bot) putQueue(msg []byte, tube string) {
	conn, err := b.Queue.Get()
	if err != nil {
		b.logger.Errorf("%+v: %s", err, string(msg))
		return
	}
	conn.Use(tube)
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
	message := tgbotapi.NewMessage(chat, msg)
	message.DisableNotification = true
	return b.Client.Send(message)
}

func (b *Bot) SendPixivIllust(target int64, id uint64) {
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⭕️", buildReactionData("pixivIllust", strconv.FormatUint(id, 10), "like")),
		tgbotapi.NewInlineKeyboardButtonData("❌", buildReactionData("pixivIllust", strconv.FormatUint(id, 10), "diss")),
	)
	msg := tgbotapi.NewMessage(target, pixiv.URLWithID(id))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(row)
	msg.DisableNotification = true
	_, err := b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func (b *Bot) startDownloadPixiv() {
	time.Sleep(10 * time.Second)
	for {
		conn, err := b.Queue.Get()
		if err != nil {
			b.logger.Errorf("%+v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		conn.Watch(tgPixivTube)
		job, err := conn.Reserve()
		if err != nil {
			b.logger.Warningf("%+v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		msg := &DownloadPixiv{}
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

		sizes, errs := pixiv.Download(msg.PixivID, b.PixivPath)
		for i := range sizes {
			if errs[i] != nil {
				err = errs[i]
				break
			}
			if sizes[i] == 0 {
				continue
			}
			b.logger.Debugf("download pixiv %d_p%d: %d bytes", msg.PixivID, i, sizes[i])
		}
		if err != nil {
			b.logger.Errorf("%+v", err)
			continue
		}

		delMsg := tgbotapi.DeleteMessageConfig{
			ChatID:    msg.ChatID,
			MessageID: msg.MessageID,
		}
		_, err = b.Client.DeleteMessage(delMsg)
		if err != nil {
			b.logger.Warningf("delete message failed: %+v", err)
		}

		err = conn.Delete(job.ID)
		if err != nil {
			b.logger.Errorf("delete job error: %+v", err)
			time.Sleep(3 * time.Second)
		}
		b.Queue.Release(conn, false)
	}
}

func (b *Bot) startDeleteMessage() {
	time.Sleep(10 * time.Second)
	for {
		conn, err := b.Queue.Get()
		if err != nil {
			b.logger.Errorf("%+v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		conn.Watch(tgDeleteTube)
		job, err := conn.Reserve()
		if err != nil {
			b.logger.Warningf("%+v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		func() {
			var err error
			defer func() {
				if err != nil {
					b.logger.Errorf("%+v", err)
					if e := conn.Bury(job.ID, 0); e != nil {
						b.logger.Errorf("%+v", err)
					}
					time.Sleep(3 * time.Second)
				} else {
					if e := conn.Delete(job.ID); e != nil {
						b.logger.Errorf("%+v", err)
						time.Sleep(3 * time.Second)
					}
				}
			}()

			msg := &tgbotapi.Message{}
			err = json.Unmarshal(job.Body, msg)
			if err != nil {
				return
			}

			if msg.Chat == nil {
				err = fmt.Errorf("err msg with no chat: %+v", msg)
				return
			}
			delMsg := tgbotapi.DeleteMessageConfig{
				ChatID:    msg.Chat.ID,
				MessageID: msg.MessageID,
			}
			b.logger.Infof("del:[%s]{%s}", getMsgTitle(msg), strconv.Quote(msg.Text))
			_, err = b.Client.DeleteMessage(delMsg)

		}()
		b.Queue.Release(conn, false)
	}
}

func (b *Bot) Start() {
	go b.startDeleteMessage()
	go b.startDownloadPixiv()

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
				case "search":
					go onReactionSearch(b, update.CallbackQuery)
				default:
				}
				continue
			} else {
				continue
			}

			if !b.checkInWhitelist(message.Chat.ID) {
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
				case "search":
					go onSearch(b, message)
				default:
					b.logger.Infof("ignore unknown cmd: %+v", message.Command())
					continue
				}
			} else {
				if message.Text == "" {
					continue
				}
				go checkRepeat(b, message)
				go checkPixiv(b, message)
				go checkSave(b, message)
			}
		}
		b.logger.Warning("tg bot restarted.")
		time.Sleep(3 * time.Second)
	}
}

func (b *Bot) checkInWhitelist(id int64) bool {
	for _, c := range b.WhitelistChats {
		if c == id {
			return true
		}
	}
	return false
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

func (b *Bot) setChatAction(chatID int64, action string) error {
	a := tgbotapi.NewChatAction(chatID, action)
	_, err := b.Client.Send(a)
	if err != nil {
		b.logger.Errorf("set action %s failed: %+v", action, err)
	}
	return err
}

func getMsgTitle(m *tgbotapi.Message) string {
	if m.Chat.IsGroup() {
		return m.Chat.Title
	}
	return m.From.String()
}
