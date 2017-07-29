package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// Config ...
type Config struct {
	File          string
	QQCfg         *QQConfig    `json:"qq"`
	RedisCfg      *RedisConfig `json:"redis"`
	BeanstalkAddr string       `json:"beanstalkAddr"`
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
}

// ReadConfig ...
func ReadConfig(cfgfile *string) (cfg *Config) {
	cfg = &Config{
		File: *cfgfile,
		QQCfg: &QQConfig{
			SendMaxRetry: 10,
		},
		RedisCfg: &RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
		BeanstalkAddr: "localhost:11300",
	}
	file, e := ioutil.ReadFile(*cfgfile)
	if e != nil {
		logger.Fatalf("Configfile error (%v)\n", e)
		os.Exit(2)
	}
	json.Unmarshal(file, cfg)
	return cfg
}
