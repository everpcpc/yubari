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
			"komatan":        "96604067",
			"maesanpicture":  "2381595966",
			"Strangestone":   "93332575",
			"kazuharukina":   "28787294",
		},
	}
	return bot
}

func hasHashTags(s string, tags []twitter.HashtagEntity) bool {
	for _, tag := range tags {
		if s == tag.Text {
			return true
		}
	}
	return false
}

func getMedias(tweet *twitter.Tweet) []twitter.MediaEntity {
	ee := tweet.ExtendedEntities
	if ee != nil {
		return ee.Media
	}
	return tweet.Entities.Media
}

func sendPics(medias []twitter.MediaEntity) {
	for _, media := range medias {
		switch media.Type {
		case "photo":
			go qqBot.SendPics(qqBot.SendGroupMsg, media.MediaURLHttps)
		}
	}
}

func logAllTrack(msg interface{}) {
	logger.Debug(msg)
}

func (t *TwitterBot) trackTweet(tweet *twitter.Tweet) {
	if tweet.RetweetedStatus != nil {
		// logger.Debugf("ignore retweet (%s):{%s}", tweet.User.Name, tweet.Text)
		return
	}
	flattenedText := strings.Replace(tweet.Text, "\n", `\n`, -1)
	medias := getMedias(tweet)
	switch tweet.User.IDStr {
	case t.Follows["KanColle_STAFF"]:
		msg := tweet.Text
		if tweet.Truncated {
			msg = tweet.FullText
		}
		logger.Infof("(%s):{%s}", tweet.User.Name, flattenedText)
		qqBot.SendGroupMsg(tweet.User.Name + "\n" + tweet.CreatedAt + "\n" + msg)
		sendPics(medias)

	case t.Follows["komatan"]:
		sendPics(medias)

	case t.Follows["maesanpicture"]:
		if !hasHashTags("毎日五月雨", tweet.Entities.Hashtags) || (len(medias) == 0) {
			logger.Debugf("(%s):{%s}", tweet.User.Name, flattenedText)
			return
		}
		logger.Infof("(%s):{%s}", tweet.User.Name, flattenedText)
		qqBot.SendGroupMsg(tweet.Text)
		sendPics(medias)

	case t.Follows["Strangestone"]:
		if !strings.HasPrefix(tweet.Text, "月曜日のたわわ") || (len(medias) == 0) {
			logger.Debugf("(%s):{%s}", tweet.User.Name, flattenedText)
			return
		}
		logger.Infof("(%s):{%s}", tweet.User.Name, flattenedText)
		qqBot.SendGroupMsg(tweet.Text)
		sendPics(medias)

	case t.Follows["kazuharukina"]:
		if !hasHashTags("和遥キナ毎日JK企画", tweet.Entities.Hashtags) || (len(medias) == 0) {
			logger.Debugf("(%s):{%s}", tweet.User.Name, flattenedText)
			return
		}
		logger.Infof("(%s):{%s}", tweet.User.Name, flattenedText)
		sendPics(medias)

	default:
		logger.Debugf("(%s):{%s}", tweet.User.Name, flattenedText)
	}
}

func (t *TwitterBot) selfProceedPics(medias []twitter.MediaEntity, action int) {
	for _, media := range medias {
		switch media.Type {
		case "photo":
			switch action {
			case 1:
				downloadFile(media.MediaURLHttps, t.ImgPath)
				go qqBot.SendPics(qqBot.SendSelfMsg, media.MediaURLHttps)
			case -1:
				removeFile(media.MediaURLHttps, t.ImgPath)
			}
		}
	}
}

func (t *TwitterBot) selfEvent(event *twitter.Event) {
	flattenedText := strings.Replace(event.TargetObject.Text, "\n", `\n`, -1)
	if event.Source.IDStr != t.ID {
		logger.Debugf("favorited: (%s):{%s}", event.Source.Name, flattenedText)
		return
	}
	switch event.Event {
	case "favorite":
		medias := getMedias(event.TargetObject)
		logger.Infof("favorite: (%s):{%s} %d medias", event.TargetObject.User.Name, flattenedText, len(medias))
		go t.selfProceedPics(medias, 1)
	case "unfavorite":
		medias := getMedias(event.TargetObject)
		logger.Debugf("unfavorite: (%s):{%s} %d medias", event.TargetObject.User.Name, flattenedText, len(medias))
		go t.selfProceedPics(medias, -1)
	default:
		logger.Debug(event.Event)
	}
}

func (t *TwitterBot) selfTweet(tweet *twitter.Tweet) {
	if qqBot.Config.NameGroup != "" {
		if hasHashTags(qqBot.Config.NameGroup, tweet.Entities.Hashtags) {
			if tweet.QuotedStatus != nil {
				logger.Infof("(%s):{%s}", qqBot.Config.NameGroup, strings.Replace(tweet.QuotedStatus.Text, "\n", `\n`, -1))
				sendPics(getMedias(tweet.QuotedStatus))
			} else {
				logger.Infof("(%s):{%s}", qqBot.Config.NameGroup, strings.Replace(tweet.Text, "\n", `\n`, -1))
				sendPics(getMedias(tweet))
			}
		}
	}
}

// Track ...
func (t *TwitterBot) Track() {
	follows := []string{}
	for _, value := range t.Follows {
		follows = append(follows, value)
	}
	for i := 1; ; i++ {
		demux := twitter.NewSwitchDemux()
		demux.Tweet = t.trackTweet
		filterParams := &twitter.StreamFilterParams{
			Follow: follows,
		}
		stream, err := t.Client.Streams.Filter(filterParams)
		if err != nil {
			logger.Error(err)
			time.Sleep(time.Duration(i) * time.Second)
		}
		demux.HandleChan(stream.Messages)
	}
}

// Self ...
func (t *TwitterBot) Self() {
	for i := 1; ; i++ {
		demux := twitter.NewSwitchDemux()
		demux.Event = t.selfEvent
		demux.Tweet = t.selfTweet
		userParams := &twitter.StreamUserParams{
			With: t.ID,
		}
		stream, err := t.Client.Streams.User(userParams)
		if err != nil {
			logger.Error(err)
			time.Sleep(time.Duration(i) * time.Second)
		}
		demux.HandleChan(stream.Messages)
	}
}
