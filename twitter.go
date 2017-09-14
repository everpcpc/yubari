package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
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
		ID:      cfg.SelfID,
		ImgPath: cfg.ImgPath,
		Client:  client,
		Follows: map[string]string{
			"KanColle_STAFF": "294025417",
			"komatan":        "96604067",
			"maesanpicture":  "2381595966",
			"Strangestone":   "93332575",
			// "kazuharukina":   "28787294",
			// "sinoalice_jp":   "818752826025181184",
			"imascg_stage": "3220191374",
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
	if tweet.ExtendedTweet != nil {
		if tweet.ExtendedTweet.ExtendedEntities != nil {
			return tweet.ExtendedTweet.ExtendedEntities.Media
		}
		return tweet.ExtendedTweet.Entities.Media
	}

	if tweet.ExtendedEntities != nil {
		return tweet.ExtendedEntities.Media
	}
	return tweet.Entities.Media
}

func sendPics(medias []twitter.MediaEntity) {
	for _, media := range medias {
		switch media.Type {
		case "photo":
			go qqBot.SendPics(qqBot.SendGroupMsg, media.MediaURLHttps)
		default:
			logger.Notice("media type ignored:", media.Type)
		}
	}
}

func logAllTrack(msg interface{}) {
	logger.Debug(msg)
}

func getTweetTime(zone string, tweet *twitter.Tweet) string {
	t := tweet.CreatedAt
	ct, err := tweet.CreatedAtTime()
	if err == nil {
		tz, err := time.LoadLocation(zone)
		if err == nil {
			t = ct.In(tz).String()
		}
	}
	return t
}

func checkSendKancolle(tweet *twitter.Tweet, msg string) {
	// sleep 5s to wait for other bot
	time.Sleep(5 * time.Second)

	ct, err := tweet.CreatedAtTime()
	if err != nil {
		logger.Error(err)
		return
	}
	key := "kancolle_" + strconv.FormatInt(ct.Unix(), 10)
	exists, err := redisClient.Expire(key, 5*time.Second).Result()
	if err != nil {
		logger.Error(err)
		return
	}
	if exists {
		logger.Notice("other bot has sent")
		return
	}

	t := getTweetTime("Asia/Tokyo", tweet)

	qqBot.SendGroupMsg(tweet.User.Name + "\n" + t + "\n\n" + msg)
}

func (t *TwitterBot) trackTweet(tweet *twitter.Tweet) {
	if tweet.RetweetedStatus != nil {
		// logger.Debugf("ignore retweet (%s):{%s}", tweet.User.Name, tweet.Text)
		return
	}
	msg := tweet.Text
	medias := getMedias(tweet)
	if tweet.Truncated {
		if tweet.ExtendedTweet != nil {
			msg = tweet.ExtendedTweet.FullText
		}
		// logger.Debugf("no ExtendedTweet: %+v", tweet)
	}
	flattenedText := strconv.Quote(msg)

	switch tweet.User.IDStr {
	case t.Follows["KanColle_STAFF"]:
		logger.Infof("(%s):{%s} %d medias", tweet.User.Name, flattenedText, len(medias))
		sendPics(medias)
		go checkSendKancolle(tweet, msg)

	case t.Follows["imascg_stage"]:
		logger.Infof("(%s):{%s} %d medias", tweet.User.Name, flattenedText, len(medias))
		t := getTweetTime("Asia/Tokyo", tweet)
		qqBot.SendGroupMsg(tweet.User.Name + "\n" + t + "\n\n" + msg)
		sendPics(medias)

	case t.Follows["komatan"]:
		if len(medias) == 0 {
			return
		}
		logger.Infof("(%s):{%s}", tweet.User.Name, flattenedText)
		sendPics(medias)

	case t.Follows["maesanpicture"]:
		if len(medias) == 0 {
			return
		}
		logger.Infof("(%s):{%s}", tweet.User.Name, flattenedText)
		if hasHashTags("毎日五月雨", tweet.Entities.Hashtags) {
			qqBot.SendGroupMsg(msg)
			sendPics(medias)
		}

	case t.Follows["Strangestone"]:
		if len(medias) == 0 {
			return
		}
		logger.Infof("(%s):{%s}", tweet.User.Name, flattenedText)
		if strings.HasPrefix(msg, "月曜日のたわわ") {
			qqBot.SendGroupMsg(msg)
			sendPics(medias)
		}

	default:
		// logger.Debugf("(%s):{%s}", tweet.User.Name, flattenedText)
	}
}

func (t *TwitterBot) selfProceedMedias(medias []twitter.MediaEntity, action int) {
	for _, media := range medias {
		switch media.Type {
		case "photo":
			switch action {
			case 1:
				file, err := downloadFile(media.MediaURLHttps, t.ImgPath)
				if err != nil {
					continue
				}
				telegramBot.sendPhoto(telegramBot.SelfChatID, file)
			case -1:
				removeFile(media.MediaURLHttps, t.ImgPath)
			}

		case "video":
			var url string
			vs := media.VideoInfo.Variants
			vsLen := len(vs)
			for i := range vs {
				if vs[vsLen-i-1].ContentType == "video/mp4" {
					url = vs[vsLen-i-1].URL
					break
				}

			}
			switch action {
			case 1:
				file, err := downloadFile(url, t.ImgPath)
				if err != nil {
					continue
				}
				telegramBot.sendVideo(telegramBot.SelfChatID, file)
			case -1:
				removeFile(url, t.ImgPath)
			}

		default:
			logger.Notice("media type ignored:", media.Type)
		}
	}
}

func (t *TwitterBot) selfEvent(event *twitter.Event) {
	if event.Source.IDStr != t.ID {
		logger.Debugf("%s: (%s)", event.Event, event.Source.Name)
		return
	}
	switch event.Event {
	case "favorite":
		medias := getMedias(event.TargetObject)
		logger.Infof("favorite: (%s):{%s} %d medias", event.TargetObject.User.Name, strconv.Quote(event.TargetObject.Text), len(medias))
		go t.selfProceedMedias(medias, 1)
	case "unfavorite":
		medias := getMedias(event.TargetObject)
		logger.Debugf("unfavorite: (%s):{%s} %d medias", event.TargetObject.User.Name, strconv.Quote(event.TargetObject.Text), len(medias))
		go t.selfProceedMedias(medias, -1)
	default:
		logger.Debug(event.Event)
	}
}

func (t *TwitterBot) selfTweet(tweet *twitter.Tweet) {
	if qqBot.Config.GroupName != "" {
		if hasHashTags(qqBot.Config.GroupName, tweet.Entities.Hashtags) {
			if tweet.QuotedStatus != nil {
				logger.Infof("(%s):{%s}", qqBot.Config.GroupName, strconv.Quote(tweet.QuotedStatus.Text))
				sendPics(getMedias(tweet.QuotedStatus))
			} else {
				logger.Infof("(%s):{%s}", qqBot.Config.GroupName, strconv.Quote(tweet.Text))
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
