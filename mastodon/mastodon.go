package mastodon

import (
	"context"

	mastodon "github.com/mattn/go-mastodon"
)

type Config struct {
	Server       string `json:"server"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	AccessToken  string `json:"accessToken"`
}

type Bot struct {
	ctx    context.Context
	client *mastodon.Client
}

func NewBot(cfg *Config) *Bot {
	client := mastodon.NewClient(&mastodon.Config{
		Server:       cfg.Server,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		AccessToken:  cfg.AccessToken,
	})
	bot := &Bot{
		ctx:    context.Background(),
		client: client,
	}
	return bot
}

func (b *Bot) NewStatus(content string, unlisted bool) {
	toot := &mastodon.Toot{Status: content}
	if unlisted {
		toot.Visibility = mastodon.VisibilityUnlisted
	}
	b.client.PostStatus(b.ctx, toot)
}
