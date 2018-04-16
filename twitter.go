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
			// "KanColle_STAFF": "294025417",
			// "imascg_stage":   "3220191374",
			// "fgoproject":     "2968069742",
			// "sinoalice_jp":   "818752826025181184",
			// "kazuharukina":   "28787294",
			"komatan":       "96604067",
			"RailgunKy":     "1269027794",
			"bison1bison":   "1557577188",
			"kentauloss":    "344418162",
			"tomo_3":        "69314676",
			"infinote":      "81257783",
			"hi_mi_tsu_2":   "803529026333642752",
			"caidychenkd":   "843223963",
			"amamitsuki12":  "958290270",
			"sakimori_st30": "718736267056263168",
			"maesanpicture": "2381595966",
			"Strangestone":  "93332575",
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

func logAllTrack(msg interface{}) {
	logger.Debugf("%+v", msg)
}

func getFullLink(tweet *twitter.Tweet) string {
	return "https://twitter.com/" + tweet.User.IDStr + "/status/" + tweet.IDStr
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
	}
	flattenedText := strconv.Quote(msg)

	switch tweet.User.IDStr {
	case t.Follows["KanColle_STAFF"], t.Follows["imascg_stage"], t.Follows["fgoproject"]:
		logger.Infof("(%s):{%s} %d medias", tweet.User.Name, flattenedText, len(medias))
		telegramBot.send(telegramBot.ChannelChatID, getFullLink(tweet))

	case t.Follows["komatan"], t.Follows["RailgunKy"], t.Follows["bison1bison"], t.Follows["kentauloss"], t.Follows["tomo_3"], t.Follows["infinote"], t.Follows["hi_mi_tsu_2"], t.Follows["caidychenkd"], t.Follows["amamitsuki12"], t.Follows["sakimori_st30"]:
		if len(medias) == 0 {
			return
		}
		logger.Infof("(%s):{%s}", tweet.User.Name, flattenedText)
		telegramBot.send(telegramBot.ChannelChatID, getFullLink(tweet))

	case t.Follows["maesanpicture"]:
		if len(medias) == 0 {
			return
		}
		logger.Infof("(%s):{%s}", tweet.User.Name, flattenedText)
		if hasHashTags("毎日五月雨", tweet.Entities.Hashtags) {
			telegramBot.send(telegramBot.ChannelChatID, getFullLink(tweet))
		}

	case t.Follows["Strangestone"]:
		if len(medias) == 0 {
			return
		}
		logger.Infof("(%s):{%s}", tweet.User.Name, flattenedText)
		if strings.HasPrefix(msg, "月曜日のたわわ") {
			telegramBot.send(telegramBot.ChannelChatID, getFullLink(tweet))
		}

	default:
		// logger.Debugf("(%s):{%s}", tweet.User.Name, flattenedText)
	}
}

func (t *TwitterBot) selfProceedMedias(medias []twitter.MediaEntity, action int) {
	var url string
	for _, media := range medias {
		switch media.Type {
		case "photo":
			url = media.MediaURLHttps

		case "video":
			vs := media.VideoInfo.Variants
			vsLen := len(vs)
			for i := range vs {
				if vs[vsLen-i-1].ContentType == "video/mp4" {
					url = vs[vsLen-i-1].URL
					break
				}
			}
		case "animated_gif":
			vs := media.VideoInfo.Variants
			vsLen := len(vs)
			for i := range vs {
				if vs[vsLen-i-1].ContentType == "video/mp4" {
					url = vs[vsLen-i-1].URL
					break
				}
			}

		default:
			logger.Noticef("media type ignored: %+v", media.Type)
			continue
		}

		switch action {
		case 1:
			downloadFile(url, t.ImgPath)
		case -1:
			removeFile(url, t.ImgPath)
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
		go telegramBot.send(telegramBot.ChannelChatID, getFullLink(event.TargetObject))
	case "unfavorite":
		medias := getMedias(event.TargetObject)
		logger.Debugf("unfavorite: (%s):{%s} %d medias", event.TargetObject.User.Name, strconv.Quote(event.TargetObject.Text), len(medias))
		go t.selfProceedMedias(medias, -1)
	default:
		logger.Debugf("%+v", event.Event)
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
			logger.Errorf("%+v", err)
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
		userParams := &twitter.StreamUserParams{
			With: t.ID,
		}
		stream, err := t.Client.Streams.User(userParams)
		if err != nil {
			logger.Errorf("%+v", err)
			time.Sleep(time.Duration(i) * time.Second)
		}
		demux.HandleChan(stream.Messages)
	}
}
