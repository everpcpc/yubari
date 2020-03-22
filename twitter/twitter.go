package twitter

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

type Config struct {
	ConsumerKey    string `json:"consumerKey"`
	ConsumerSecret string `json:"consumerSecret"`
	AccessToken    string `json:"accessToken"`
	AccessSecret   string `json:"accessSecret"`
	SelfID         string `json:"selfID"`
	ImgPath        string `json:"imgPath"`
}

type Bot struct {
	ID      string
	ImgPath string
	Client  *twitter.Client
	Follows map[string]string
}

func NewBot(cfg *Config) *Bot {
	config := oauth1.NewConfig(cfg.ConsumerKey, cfg.ConsumerSecret)
	token := oauth1.NewToken(cfg.AccessToken, cfg.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)
	bot := &Bot{
		ID:      cfg.SelfID,
		ImgPath: cfg.ImgPath,
		Client:  client,
		Follows: map[string]string{
			"KanColle_STAFF": "294025417",
			// "imascg_stage":   "3220191374",
			// "fgoproject":     "2968069742",
			// "sinoalice_jp":   "818752826025181184",
			// "kazuharukina":   "28787294",

			// "komatan":       "96604067",
			// "RailgunKy":     "1269027794",
			// "bison1bison":   "1557577188",
			// "kentauloss":    "344418162",
			// "tomo_3":        "69314676",
			// "infinote":      "81257783",
			// "hi_mi_tsu_2":   "803529026333642752",
			// "caidychenkd":   "843223963",
			// "amamitsuki12":  "958290270",
			// "sakimori_st30": "718736267056263168",
			// "maesanpicture": "2381595966",
			// "Strangestone":  "93332575",
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

func getFullLink(tweet *twitter.Tweet) string {
	return "https://twitter.com/" + tweet.User.IDStr + "/status/" + tweet.IDStr
}
