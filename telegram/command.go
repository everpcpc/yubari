package telegram

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"yubari/pixiv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gopkg.in/gographics/imagick.v3/imagick"
)

func onStart(b *Bot, message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "呀呀呀")
	msg.ReplyToMessageID = message.MessageID
	msg.DisableNotification = true

	b.Client.Send(msg)
}

func onRoll(b *Bot, message *tgbotapi.Message) {
	b.setChatAction(message.Chat.ID, "typing")

	var (
		err   error
		limit int
	)

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
	msg.DisableNotification = true

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%+v", err)
	}
}

func onComic(b *Bot, message *tgbotapi.Message) {
	b.setChatAction(message.Chat.ID, "typing")

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

	msg.ReplyMarkup = buildLikeButton(b.redis, "comic", number)
	msg.DisableNotification = true

	b.logger.Infof("send comic:[%s]{%s}", getMsgTitle(message), strconv.Quote(file))
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
	b.putQueue(data, tgDeleteTube)
}

func onPixivWithArgs(args string, b *Bot, message *tgbotapi.Message) {
	if id, err := strconv.ParseUint(args, 10, 0); err == nil {
		b.SendPixivCandidate(message.Chat.ID, id)
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
	b.putQueue(data, tgDeleteTube)
}

func onPixivNoArgs(b *Bot, message *tgbotapi.Message) {
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
	filePath := files[rand.Intn(len(files))]
	fileName := filepath.Base(filePath)

	b.logger.Infof("send pixiv:[%s]{%s}", getMsgTitle(message), strconv.Quote(filePath))

	pid, err := strconv.ParseUint(strings.Split(fileName, "_")[0], 10, 0)
	if err != nil {
		b.logger.Errorf("parse pid from file name failed: %+v", err)
		return
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	err = mw.ReadImage(filePath)
	if err != nil {
		b.logger.Errorf("read image failed: %+v", err)
		return
	}
	width := mw.GetImageWidth()
	height := mw.GetImageHeight()

	err = mw.ResizeImage(640, 640*height/width, 0)
	if err != nil {
		b.logger.Errorf("resize image failed: %+v", err)
		return
	}

	msg := tgbotapi.NewPhotoUpload(message.Chat.ID, tgbotapi.FileBytes{
		Name:  fileName,
		Bytes: mw.GetImageBlob(),
	})
	msg.Caption = fmt.Sprintf(
		"<a href=\"%s\">pixiv:%d</a>(%dx%d)",
		pixiv.URLWithID(pid), pid, width, height,
	)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = buildLikeButton(b.redis, "pixiv", fileName)
	msg.ReplyToMessageID = message.MessageID
	msg.DisableNotification = true

	b.setChatAction(message.Chat.ID, "upload_photo")

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("send pixiv failed: %+v", err)
	}
}
