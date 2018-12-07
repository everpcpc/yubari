package main

import (
	"encoding/json"
	"io/ioutil"
)

// Config ...
type Config struct {
	File          string
	BeanstalkAddr string          `json:"beanstalkAddr"`
	Redis         *RedisConfig    `json:"redis"`
	QQ            *QQConfig       `json:"qq"`
	Twitter       *TwitterConfig  `json:"twitter"`
	Telegram      *TelegramConfig `json:"telegram"`
	Pixiv         *PixivConfig    `json:"pixiv"`
	BgmID         string          `json:"bgmID"`
	SentryDSN     string          `json:"sentry"`
}

// RedisConfig ...
type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// QQConfig ...
type QQConfig struct {
	SelfID          string   `json:"selfID"`
	BotID           string   `json:"botID"`
	QQPrivateIgnore []string `json:"qqPrivateIgnore"`
	QQGroupIgnore   []string `json:"qqGroupIgnore"`
	SelfNames       []string `json:"selfNames"`
	SendMaxRetry    int      `json:"sendMaxRetry"`
	ImgPath         string   `json:"imgPath"`
}

// TwitterConfig ...
type TwitterConfig struct {
	ConsumerKey    string `json:"consumerKey"`
	ConsumerSecret string `json:"consumerSecret"`
	AccessToken    string `json:"accessToken"`
	AccessSecret   string `json:"accessSecret"`
	SelfID         string `json:"selfID"`
	ImgPath        string `json:"imgPath"`
}

// TelegramConfig ...
type TelegramConfig struct {
	Token          string  `json:"token"`
	SelfID         int64   `json:"selfID"`
	WhitelistChats []int64 `json:"whitelistChats"`
	ComicPath      string  `json:"comicPath"`
	DeleteDelay    string  `json:"deleteDelay"`
}

// PixivConfig ...
type PixivConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	ImgPath  string `json:"imgPath"`
}

// ReadConfig ...
func ReadConfig(cfgfile *string) (cfg *Config) {
	cfg = &Config{
		File:          *cfgfile,
		BeanstalkAddr: "localhost:11300",
		QQ: &QQConfig{
			SendMaxRetry: 10,
		},
		Redis: &RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
		Twitter:  &TwitterConfig{},
		Telegram: &TelegramConfig{},
	}
	file, e := ioutil.ReadFile(*cfgfile)
	if e != nil {
		logger.Fatalf("Configfile error (%v)\n", e)
	}
	json.Unmarshal(file, cfg)
	return cfg
}
