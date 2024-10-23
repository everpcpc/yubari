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

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
				b.logger.Errorf("%s", err)
			}
			return
		}
	}
	if limit <= 0 {
		limit = 100
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	msg := tgbotapi.NewMessage(message.Chat.ID, "🎲 "+strconv.Itoa(r.Intn(limit)))
	msg.ReplyToMessageID = message.MessageID
	msg.DisableNotification = true

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%s", err)
	}
}

func onComic(b *Bot, message *tgbotapi.Message) {
	b.setChatAction(message.Chat.ID, "typing")

	files, err := filepath.Glob(filepath.Join(b.ComicPath, "*.epub"))
	if err != nil {
		b.logger.Errorf("%s", err)
		return
	}
	if files == nil {
		b.logger.Error("find no comic")
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	file := files[r.Intn(len(files))]
	number := strings.Split(strings.Split(file, "@")[1], ".")[0]
	msg := tgbotapi.NewMessage(message.Chat.ID, "🔞 https://nhentai.net/g/"+number)

	msg.ReplyMarkup = buildLikeButton(b.redis, "comic", number)
	msg.DisableNotification = true

	b.logger.Infof("send comic:[%s]{%s}", getMsgTitle(message), strconv.Quote(file))
	msgSent, err := b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%s", err)
		return
	}
	data, err := json.Marshal(msgSent)
	if err != nil {
		b.logger.Errorf("%s", err)
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
		b.logger.Errorf("%s", err)
		return
	}
	data, err := json.Marshal(msgSent)
	if err != nil {
		b.logger.Errorf("%s", err)
		return
	}
	b.putQueue(data, tgDeleteTube)
}

func onPixivNoArgs(b *Bot, message *tgbotapi.Message) {
	filePath, err := b.pixivBot.RandomPic()
	if err != nil {
		b.logger.Errorf("random pixiv error: %s", err)
		return
	}
	fileName := filepath.Base(filePath)

	b.logger.Infof("send pixiv:[%s]{%s}", getMsgTitle(message), strconv.Quote(filePath))

	pid, err := strconv.ParseUint(strings.Split(fileName, "_")[0], 10, 0)
	if err != nil {
		b.logger.Errorf("parse pid from file name failed: %s", err)
		return
	}
	illust, err := b.pixivBot.Get(pid)
	if err != nil {
		b.logger.Errorf("get pixiv illust failed: %s", err)
		return
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	err = mw.ReadImage(filePath)
	if err != nil {
		b.logger.Errorf("read image failed: %s", err)
		return
	}
	width := mw.GetImageWidth()
	height := mw.GetImageHeight()

	err = mw.ResizeImage(640, 640*height/width, imagick.FILTER_BOX)
	if err != nil {
		b.logger.Errorf("resize image failed: %s", err)
		return
	}

	blob, err := mw.GetImageBlob()
	if err != nil {
		b.logger.Errorf("get image blob failed: %s", err)
		return
	}
	msg := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FileBytes{
		Name:  fileName,
		Bytes: blob,
	})
	msg.ParseMode = tgbotapi.ModeHTML
	tags := ""
	for _, tag := range illust.Tags {
		tags += fmt.Sprintf("#%s ", tag.Name)
	}
	msg.Caption = fmt.Sprintf(
		"<a href=\"%s\">%s: %s</a>\n%s",
		pixiv.URLWithID(pid),
		illust.User.Name, illust.Title,
		tags,
	)
	msg.ReplyMarkup = buildLikeButton(b.redis, "pixiv", filePath)
	msg.ReplyToMessageID = message.MessageID
	msg.DisableNotification = true

	b.setChatAction(message.Chat.ID, "upload_photo")

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("send pixiv failed: %s", err)
	}
}
