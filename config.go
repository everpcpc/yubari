package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/everpcpc/yubari/pixiv"
	"github.com/everpcpc/yubari/qq"
	"github.com/everpcpc/yubari/telegram"
	"github.com/everpcpc/yubari/twitter"
)

type Config struct {
	File          string
	BeanstalkAddr string           `json:"beanstalkAddr"`
	Redis         *RedisConfig     `json:"redis"`
	QQ            *qq.Config       `json:"qq"`
	Twitter       *twitter.Config  `json:"twitter"`
	Telegram      *telegram.Config `json:"telegram"`
	Pixiv         *pixiv.Config    `json:"pixiv"`
	BgmID         string           `json:"bgmID"`
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
		QQ: &qq.Config{
			SendMaxRetry: 10,
		},
		Redis: &RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
		Twitter:  &twitter.Config{},
		Telegram: &telegram.Config{},
	}
	file, e := ioutil.ReadFile(*cfgfile)
	if e != nil {
		logger.Fatalf("Configfile error (%v)\n", e)
	}
	json.Unmarshal(file, cfg)
	return cfg
}
