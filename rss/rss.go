package rss

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
)

type Feed struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	TTL  int    `json:"ttl"`
}

func (f *Feed) getLastKey() string {
	return fmt.Sprintf("rss_last_%s_%s", f.Type, f.URL)
}

func (f *Feed) String() string {
	return fmt.Sprintf("feed<%s:%ds:%s>", f.Type, f.TTL, f.URL)
}

type Config struct {
	Feeds []*Feed `json:"feeds"`
}

type Bot struct {
	ctx    context.Context
	output chan string
	feeds  []*Feed
	redis  *redis.Client
	logger *logrus.Logger
}

func NewBot(cfg *Config, output chan string) *Bot {
	b := &Bot{
		ctx:    context.Background(),
		output: output,
		feeds:  cfg.Feeds,
	}
	return b
}

func (b *Bot) WithLogger(logger *logrus.Logger) *Bot {
	b.logger = logger
	return b
}

func (b *Bot) WithRedis(rds *redis.Client) *Bot {
	b.redis = rds
	return b
}

func (b *Bot) trackFeed(feed *Feed) {
	lastKey := feed.getLastKey()
	ticker := time.NewTicker(time.Duration(feed.TTL) * time.Second)
	fp := gofeed.NewParser()
	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.logger.Debugf("checking rss: %s", feed)
			msg, err := fp.ParseURL(feed.URL)
			if err != nil {
				b.logger.Errorf("%+v", err)
				time.Sleep(time.Second)
				continue
			}
			last, err := b.redis.Get(lastKey).Int64()
			if err != nil {
				b.logger.Warningf("get last error: %+v", err)
				last = 0
			}
			var latest int64
			for _, item := range msg.Items {
				if item.GUID == "" {
					b.logger.Errorf("guid not found for %+v", item.Title)
					continue
				}
				tokens := strings.Split(item.GUID, "/")
				guid := tokens[len(tokens)-1]
				id, err := strconv.ParseInt(guid, 10, 64)
				if err != nil {
					b.logger.Errorf("guid: %+v", item.GUID)
					continue
				}
				// latest: largest id in feed items
				if id > latest {
					latest = id
				}
				// first feed
				if last == 0 {
					last = id
					break
				}
				// older item
				if id <= last {
					break
				}
				text, err := getBangumiUpdate(item)
				if err != nil {
					b.logger.Errorf("%+v", err)
					time.Sleep(time.Second)
					continue
				}
				b.logger.Infof("rss: %s", text)
				b.output <- text
			}
			if b.redis.Set(lastKey, latest, 0).Err() != nil {
				b.logger.Errorf("set last %+v", err)
			}
		}
	}

}

func (b *Bot) Start() {
	for _, feed := range b.feeds {
		go b.trackFeed(feed)
	}
}
