package telegram

import (
	"encoding/json"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

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

	msg.ReplyMarkup = buildInlineKeyboardMarkup(b.redis, "comic", number)

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
	msg.ReplyMarkup = buildInlineKeyboardMarkup(b.redis, "pic", filepath.Base(file))

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
	msg.ReplyMarkup = buildInlineKeyboardMarkup(b.redis, "pixiv", filepath.Base(file))
	msg.ReplyToMessageID = message.MessageID

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func onSearch(b *Bot, message *tgbotapi.Message) {
	args := message.CommandArguments()
	msg := tgbotapi.NewMessage(message.Chat.ID, "呀呀呀"+args)

	msg.ReplyToMessageID = message.MessageID
	b.Client.Send(msg)
}
