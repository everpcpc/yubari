package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// Config ...
type Config struct {
	File          string
	BeanstalkAddr string         `json:"beanstalkAddr"`
	Redis         *RedisConfig   `json:"redis"`
	QQ            *QQConfig      `json:"qq"`
	Twitter       *TwitterConfig `json:"twitter"`
}

// RedisConfig ...
type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// QQConfig ...
type QQConfig struct {
	QQSelf          string   `json:"qqSelf"`
	QQBot           string   `json:"qqBot"`
	QQGroup         string   `json:"qqGroup"`
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
	IDSelf         string `json:"idSelf"`
	ImgPath        string `json:"imgPath"`
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
		Twitter: &TwitterConfig{},
	}
	file, e := ioutil.ReadFile(*cfgfile)
	if e != nil {
		logger.Fatalf("Configfile error (%v)\n", e)
		os.Exit(2)
	}
	json.Unmarshal(file, cfg)
	return cfg
}
