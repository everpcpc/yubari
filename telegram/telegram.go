package telegram

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	bt "github.com/ikool-cn/gobeanstalk-connection-pool"
	meilisearch "github.com/meilisearch/meilisearch-go"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	"yubari/pixiv"
)

const (
	tgDeleteTube = "tg_delete"
	tgPixivTube  = "tg_pixiv"

	pixivSplitWidth = 10000000
)

var (
	rePixivFileName = regexp.MustCompile(`(?P<id>\d+)_p(?P<seq>d+)\.(?P<ext>\w+)`)
)

type Config struct {
	Token          string  `json:"token"`
	SelfID         int64   `json:"selfID"`
	AdmissionID    int64   `json:"admissionID"`
	WhitelistChats []int64 `json:"whitelistChats"`
	ComicPath      string  `json:"comicPath"`
	DeleteDelay    string  `json:"deleteDelay"`
	OpenAIKey      string  `json:"openAIKey"`
}

type DownloadPixiv struct {
	ChatID    int64
	MessageID int
	PixivID   uint64
	Text      string
}

type Bot struct {
	Name           string
	SelfID         int64
	AdmissionID    int64
	WhitelistChats []int64
	ComicPath      string

	DeleteDelay time.Duration
	Client      *tgbotapi.BotAPI
	Queue       *bt.Pool
	logger      *logrus.Logger
	redis       *redis.Client
	meili       *meilisearch.Client

	pixivBot *pixiv.Bot
	ai       *openai.Client
}

func NewBot(cfg *Config) (b *Bot, err error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("tg bot init failed: %s", err)
	}
	delay, err := time.ParseDuration(cfg.DeleteDelay)
	if err != nil {
		return nil, fmt.Errorf("delete delay error: %s", err)
	}
	b = &Bot{
		Name:           bot.Self.UserName,
		SelfID:         cfg.SelfID,
		AdmissionID:    cfg.AdmissionID,
		WhitelistChats: cfg.WhitelistChats,
		ComicPath:      cfg.ComicPath,
		DeleteDelay:    delay,
		Client:         bot,
	}

	return
}

func (b *Bot) WithLogger(logger *logrus.Logger) *Bot {
	b.logger = logger
	return b
}

func (b *Bot) WithRedis(rds *redis.Client) *Bot {
	b.redis = rds
	return b
}

func (b *Bot) WithPixiv(bot *pixiv.Bot) *Bot {
	b.pixivBot = bot
	return b
}

func (b *Bot) WithQueue(queue *bt.Pool) *Bot {
	b.Queue = queue
	return b
}

func (b *Bot) WithMeilisearch(meili *meilisearch.Client) *Bot {
	b.meili = meili
	return b
}

func (b *Bot) WithOpenAI(key string) *Bot {
	b.ai = openai.NewClient(key)
	return b
}

func (b *Bot) putQueue(msg []byte, tube string) {
	conn, err := b.Queue.Get()
	if err != nil {
		b.logger.Errorf("%s: %s", err, string(msg))
		return
	}
	conn.Use(tube)
	_, err = conn.Put(msg, 1, b.DeleteDelay, time.Minute)
	if err != nil {
		b.logger.Errorf("%s", err)
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

func (b *Bot) GetUserName(chatID int64, userID int64) (name string, err error) {
	cacheKey := fmt.Sprintf("tg:user:%d", userID)
	cache, err := b.redis.Get(cacheKey).Result()
	if err == nil {
		name = cache
		return
	} else {
		if err != redis.Nil {
			return
		}
	}
	member, err := b.Client.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatID,
			UserID: userID,
		},
	})
	if err != nil {
		return
	}
	name = member.User.String()
	if name != "" {
		b.redis.Set(cacheKey, name, 0)
	}
	return
}

func (b *Bot) getIndex(message *tgbotapi.Message) *meilisearch.Index {
	return b.meili.Index(fmt.Sprintf("%s-%d", message.Chat.Type, message.Chat.ID))
}

func (b *Bot) SendPixivCandidate(target int64, id uint64) {
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⭕️", buildReactionData("pixivCandidate", strconv.FormatUint(id, 10), "like")),
		tgbotapi.NewInlineKeyboardButtonData("❌", buildReactionData("pixivCandidate", strconv.FormatUint(id, 10), "diss")),
	)
	msg := tgbotapi.NewMessage(target, pixiv.URLWithID(id))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(row)
	msg.DisableNotification = true
	_, err := b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%s", err)
	}
}

func (b *Bot) startDownloadPixiv() {
	time.Sleep(10 * time.Second)
	for {
		conn, err := b.Queue.Dial()
		if err != nil {
			b.logger.Errorf("%s", err)
			time.Sleep(3 * time.Second)
			continue
		}
		conn.Watch(tgPixivTube)
		job, err := conn.Reserve(60 * time.Second)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		msg := &DownloadPixiv{}
		err = json.Unmarshal(job.Body, msg)
		if err != nil {
			b.logger.Errorf("%s", err)
			err = conn.Bury(job.ID, 0)
			if err != nil {
				b.logger.Errorf("%s", err)
			}
			time.Sleep(3 * time.Second)
			continue
		}

		sizes, err := b.pixivBot.Download(msg.PixivID)
		if err != nil {
			b.logger.Errorf("failed downloading pixiv %d: %s", msg.PixivID, err)
			conn.Release(job.ID, 0, 10*time.Second)
			b.Queue.Release(conn, false)
			continue
		}

		for i := range sizes {
			if sizes[i] == 0 {
				b.logger.Debugf("pixiv %d_p%d: exists", msg.PixivID, i)
				msg.Text += fmt.Sprintf("\n🈶 p%d", i)
			} else {
				b.logger.Debugf("download pixiv %d_p%d: %s", msg.PixivID, i, ByteCountIEC(sizes[i]))
				msg.Text += fmt.Sprintf("\n✅ p%d - %s", i, ByteCountIEC(sizes[i]))
			}
		}

		updateTextMsg := tgbotapi.NewEditMessageText(
			msg.ChatID,
			msg.MessageID,
			msg.Text,
		)
		updateTextMsg.DisableWebPagePreview = true
		_, err = b.Client.Send(updateTextMsg)
		if err != nil {
			b.logger.Errorf("error update message text %s", err)
		}

		err = conn.Delete(job.ID)
		if err != nil {
			b.logger.Errorf("delete job error: %s", err)
			time.Sleep(3 * time.Second)
		}
		b.Queue.Release(conn, false)
	}
}

func (b *Bot) startDeleteMessage() {
	time.Sleep(10 * time.Second)
	for {
		conn, err := b.Queue.Dial()
		if err != nil {
			b.logger.Errorf("%s", err)
			time.Sleep(3 * time.Second)
			continue
		}
		conn.Watch(tgDeleteTube)
		job, err := conn.Reserve(60 * time.Second)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		func() {
			var err error
			defer func() {
				if err != nil {
					b.logger.Errorf("%s", err)
					if e := conn.Bury(job.ID, 0); e != nil {
						b.logger.Errorf("%s", err)
					}
					time.Sleep(3 * time.Second)
				} else {
					if e := conn.Delete(job.ID); e != nil {
						b.logger.Errorf("%s", err)
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
			b.logger.Infof("del:[%s]{%s}", getMsgTitle(msg), strconv.Quote(msg.Text))
			_, err = b.Client.Send(tgbotapi.DeleteMessageConfig{
				ChatID:    msg.Chat.ID,
				MessageID: msg.MessageID,
			})

		}()
		b.Queue.Release(conn, false)
	}
}

func (b *Bot) Start() {
	go b.startDeleteMessage()
	go b.startDownloadPixiv()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := b.Client.GetUpdatesChan(u)
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
			case "pixivCandidate":
				if !b.isAuthedChat(update.CallbackQuery.Message.Chat) {
					b.logger.Warning("reaction from illegal chat, ignore")
					break
				}
				go onReactionCandidate(b, update.CallbackQuery)
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
			case "pixiv":
				args := message.CommandArguments()
				if args != "" {
					go onPixivWithArgs(args, b, message)
				} else {
					go onPixivNoArgs(b, message)
				}
			case "search":
				go onSearch(b, message)
			default:
				b.logger.Infof("ignore unknown cmd: %s", message.Command())
				continue
			}
		} else {
			if message.Text == "" {
				continue
			}
			go checkRepeat(b, message)
			go checkPixiv(b, message)
			go checkSave(b, message)
			go checkOpenAI(b, message)
		}
	}
	b.logger.Warning("tg bot restarted.")
	time.Sleep(3 * time.Second)
}

func (b *Bot) checkInWhitelist(id int64) bool {
	for _, c := range b.WhitelistChats {
		if c == id {
			return true
		}
	}
	b.logger.Debugf("ignore msg from %d", id)
	return false
}

func (b *Bot) probate(_type, _target string) error {
	b.logger.Infof("%s: %s", _type, _target)
	switch _type {
	case "comic":
		fileName := "nhentai.net@" + _target + ".epub"
		return os.Rename(
			filepath.Join(b.ComicPath, fileName),
			filepath.Join(b.ComicPath, "probation", fileName),
		)
	case "pixiv":
		return b.pixivBot.Probate(_target)
	default:
		return fmt.Errorf("prohibit unkown type")
	}
}

func (b *Bot) setChatAction(chatID int64, action string) error {
	a := tgbotapi.NewChatAction(chatID, action)
	_, err := b.Client.Request(a)
	if err != nil {
		b.logger.Errorf("set action %s failed: %s", action, err)
	}
	return err
}

func getMsgTitle(m *tgbotapi.Message) string {
	if m.Chat.IsGroup() {
		return m.Chat.Title
	}
	return m.From.String()
}
