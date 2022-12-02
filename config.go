package main

import (
	"encoding/json"
	"os"

	"yubari/mastodon"
	"yubari/meili"
	"yubari/pixiv"
	"yubari/rss"
	"yubari/telegram"
)

type Config struct {
	File          string
	BeanstalkAddr string           `json:"beanstalkAddr"`
	Redis         *RedisConfig     `json:"redis"`
	Mastodon      *mastodon.Config `json:"mastodon"`
	Telegram      *telegram.Config `json:"telegram"`
	Pixiv         *pixiv.Config    `json:"pixiv"`
	RSS           *rss.Config      `json:"rss"`
	Meilisearch   *meili.Config    `json:"meilisearch"`
	SentryDSN     string           `json:"sentry"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

func ReadConfig(cfgfile *string) (cfg *Config) {
	cfg = &Config{
		File:          *cfgfile,
		BeanstalkAddr: "localhost:11300",
		Redis: &RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
		Telegram:    &telegram.Config{},
		Meilisearch: &meili.Config{},
	}
	file, e := os.ReadFile(*cfgfile)
	if e != nil {
		logger.Fatalf("Configfile error (%v)\n", e)
	}
	json.Unmarshal(file, cfg)
	return cfg
}
