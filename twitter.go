package main

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"strings"
	"time"
)

var (
	twitterBot *TwitterBot
)

// TwitterBot ...
type TwitterBot struct {
	ID      string
	ImgPath string
	Client  *twitter.Client
	Follows map[string]string
}

// NewTwitterBot ...
func NewTwitterBot(cfg *TwitterConfig) *TwitterBot {
	config := oauth1.NewConfig(cfg.ConsumerKey, cfg.ConsumerSecret)
	token := oauth1.NewToken(cfg.AccessToken, cfg.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)
	bot := &TwitterBot{
		ID:      cfg.IDSelf,
		ImgPath: cfg.ImgPath,
		Client:  client,
		Follows: map[string]string{
			"KanColle_STAFF": "294025417",
			"maesanpicture":  "2381595966",
			"komatan":        "96604067",
			"Strangestone":   "93332575",
		},
	}
	return bot
}

func logAllTrack(msg interface{}) {
	logger.Debug(msg)
}

func trackSendPics(medias []twitter.MediaEntity) {
	for _, media := range medias {
		switch media.Type {
		case "photo":
			fileName, err := downloadFile(media.MediaURLHttps, qqBot.Config.ImgPath)
			if err != nil {
				continue
			}
			qqBot.SendGroupMsg(QQImage{fileName}.String())
		}
	}
}

func hasHashTags(s string, tags []twitter.HashtagEntity) bool {
	for _, tag := range tags {
		if s == tag.Text {
			return true
		}
	}
	return false
}

func proceedTrack(tweet *twitter.Tweet) {
	switch tweet.User.IDStr {
	case twitterBot.Follows["KanColle_STAFF"]:
		medias := getMedias(tweet)
		trackSendPics(medias)
		logger.Infof("%s: {%s}", tweet.User.Name, strings.Replace(tweet.Text, "\n", " ", -1))
	case twitterBot.Follows["maesanpicture"]:
		logger.Debugf("%s: {%s}", tweet.User.Name, strings.Replace(tweet.Text, "\n", " ", -1))
		if !hasHashTags("毎日五月雨", tweet.Entities.Hashtags) {
			return
		}
		medias := getMedias(tweet)
		if len(medias) == 0 {
			return
		}
		qqBot.SendGroupMsg(tweet.Text)
		trackSendPics(medias)
		logger.Infof("%s: {%s}", tweet.User.Name, strings.Replace(tweet.Text, "\n", " ", -1))
	case twitterBot.Follows["komatan"]:
		medias := getMedias(tweet)
		trackSendPics(medias)
		logger.Infof("%s: {%s}", tweet.User.Name, strings.Replace(tweet.Text, "\n", " ", -1))
	case twitterBot.Follows["Strangestone"]:
		logger.Debugf("%s: {%s}", tweet.User.Name, strings.Replace(tweet.Text, "\n", " ", -1))
		if !strings.HasPrefix("月曜日のたわわ", tweet.Text) {
			return
		}
		medias := getMedias(tweet)
		if len(medias) == 0 {
			return
		}
		qqBot.SendGroupMsg(tweet.Text)
		trackSendPics(medias)
		logger.Infof("%s: {%s}", tweet.User.Name, strings.Replace(tweet.Text, "\n", " ", -1))
	default:
		logger.Debugf("(%s):{%s}", tweet.User.IDStr, tweet.Text)
	}
}

func getMedias(tweet *twitter.Tweet) []twitter.MediaEntity {
	medias := tweet.ExtendedEntities.Media
	if len(medias) == 0 {
		medias = tweet.Entities.Media
	}
	return medias
}

func selfProceedPics(medias []twitter.MediaEntity, action int) {
	for _, media := range medias {
		switch media.Type {
		case "photo":
			switch action {
			case 1:
				downloadFile(media.MediaURLHttps, twitterBot.ImgPath)
			case -1:
				removeFile(media.MediaURLHttps, twitterBot.ImgPath)
			}
		}
	}
}

func eventSelf(event *twitter.Event) {
	switch event.Event {
	case "favorite":
		medias := getMedias(event.TargetObject)
		logger.Infof("favorite: [%s] %d medias", strings.Replace(event.TargetObject.Text, "\n", " ", -1), len(medias))
		go selfProceedPics(medias, 1)
	case "unfavorite":
		medias := getMedias(event.TargetObject)
		logger.Debugf("unfavorite: [%s] %d medias", strings.Replace(event.TargetObject.Text, "\n", " ", -1), len(medias))
		go selfProceedPics(medias, -1)
	default:
		logger.Debug(event.Event)
	}
}

func twitterTrack() {
	follows := []string{}
	for _, value := range twitterBot.Follows {
		follows = append(follows, value)
	}
	for i := 1; ; i++ {
		demux := twitter.NewSwitchDemux()
		demux.Tweet = proceedTrack
		filterParams := &twitter.StreamFilterParams{
			Follow: follows,
		}
		stream, err := twitterBot.Client.Streams.Filter(filterParams)
		if err != nil {
			logger.Error(err)
			time.Sleep(time.Duration(i) * time.Second)
		}
		demux.HandleChan(stream.Messages)
	}
}

func twitterSelf() {
	for i := 1; ; i++ {
		demux := twitter.NewSwitchDemux()
		demux.Event = eventSelf
		userParams := &twitter.StreamUserParams{
			With: twitterBot.ID,
		}
		stream, err := twitterBot.Client.Streams.User(userParams)
		if err != nil {
			logger.Error(err)
			time.Sleep(time.Duration(i) * time.Second)
		}
		demux.HandleChan(stream.Messages)
	}
}
