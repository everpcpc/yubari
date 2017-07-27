package main

import (
	"flag"
)

func main() {
	logger = GetLogger(LOGSTD)

	cfgfile := flag.String("c", "config.json", "Config file")
	flag.Parse()
	cfg := ReadConfig(cfgfile)
	logger.Infof("Starting with config: %+v", cfg)

	var err error
	rds, err = NewRedisClient(cfg)
	if err != nil {
		logger.Panic(err)
		return
	}
	logger.Infof("Redis connected: %+v", rds)
	qqBot, err = NewQQBot(cfg)
	if err != nil {
		logger.Panic(err)
		return
	}
	logger.Infof("Starting qqbot: %+v", qqBot)
	qqWatch()
}

func qqWatch() {
	messages := make(chan map[string]string)
	go qqBot.Poll(messages)
	for msg := range messages {
		switch msg["event"] {
		case "PrivateMsg":
			logger.Infof("[%s]:{%s}", msg["qq"], msg["msg"])
		case "GroupMsg":
			logger.Infof("(%s)[%s]:{%s}", msg["group"], msg["qq"], msg["msg"])
		default:
			logger.Info(msg)
		}
	}
}
