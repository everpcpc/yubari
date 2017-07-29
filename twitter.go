package main

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

var (
	twitterClient *twitter.Client
)

// NewTwitterClient ...
func NewTwitterClient(cfg *TwitterConfig) *twitter.Client {
	config := oauth1.NewConfig(cfg.ConsumerKey, cfg.ConsumerSecret)
	token := oauth1.NewToken(cfg.AccessToken, cfg.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)
	return client
}

func twitterTrack() {
}

func twitterPics() {
}
