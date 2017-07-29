package main

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

var (
	twitterBot *TwitterBot
)

// TwitterBot ...
type TwitterBot struct {
	Client *twitter.Client
}

// NewTwitterBot ...
func NewTwitterBot(cfg *TwitterConfig) *TwitterBot {
	config := oauth1.NewConfig(cfg.ConsumerKey, cfg.ConsumerSecret)
	token := oauth1.NewToken(cfg.AccessToken, cfg.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)
	bot := &TwitterBot{
		Client: client,
	}
	return bot
}

func proceedTweet(tweet *twitter.Tweet) {
}

func downloadFavPics(event *twitter.Event) {
	switch event.Event {
	case "favorite":
	case "unfavorite":
	default:
	}
}

func twitterTrack() {
	demux := twitter.NewSwitchDemux()
	demux.Tweet = proceedTweet
}

func twitterPics() {
	demux := twitter.NewSwitchDemux()
	demux.Event = downloadFavPics
}
