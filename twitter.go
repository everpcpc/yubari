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
	}
	return bot
}

func proceedTweet(tweet *twitter.Tweet) {
}

func getMedias(tweet *twitter.Tweet) []twitter.MediaEntity {
	medias := tweet.ExtendedEntities.Media
	if len(medias) == 0 {
		medias = tweet.Entities.Media
	}
	return medias
}

func proceedPics(medias []twitter.MediaEntity, action int) {
	for _, media := range medias {
		switch media.Type {
		case "photo":
			switch action {
			case 1:
				downloadFile(media.MediaURLHttps, twitterBot.ImgPath)
			case -1:
				removeFile(media.MediaURLHttps, twitterBot.ImgPath)
			default:
			}
		default:
		}
	}
}

func eventSelf(event *twitter.Event) {
	switch event.Event {
	case "favorite":
		medias := getMedias(event.TargetObject)
		logger.Infof("favorite: [%s] %d medias", strings.Replace(event.TargetObject.Text, "\n", " ", -1), len(medias))
		go proceedPics(medias, 1)
	case "unfavorite":
		medias := getMedias(event.TargetObject)
		logger.Debugf("unfavorite: [%s] %d medias", strings.Replace(event.TargetObject.Text, "\n", " ", -1), len(medias))
		go proceedPics(medias, -1)
	default:
		logger.Debug(event.Event)
	}
}

func twitterTrack() {
	follows := []string{
		"294025417",  //KanColle_STAFF"
		"2381595966", //maesanpicture
		"96604067",   //komatan
		"93332575",   //Strangestone
	}

	for i := 1; ; i++ {
		demux := twitter.NewSwitchDemux()
		demux.Tweet = proceedTweet
		filterParams := &twitter.StreamFilterParams{
			Follow:        follows,
			StallWarnings: twitter.Bool(true),
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
			With:          twitterBot.ID,
			StallWarnings: twitter.Bool(true),
		}
		stream, err := twitterBot.Client.Streams.User(userParams)
		if err != nil {
			logger.Error(err)
			time.Sleep(time.Duration(i) * time.Second)
		}
		demux.HandleChan(stream.Messages)
	}
}
