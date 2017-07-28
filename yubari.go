package main

import (
	"flag"
	"time"
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
	defer rds.Close()
	logger.Infof("Redis connected: %+v", rds)

	qqBot = NewQQBot(cfg)
	defer qqBot.Pool.Close()
	logger.Infof("QQBot: %+v", qqBot)

	go qqWatch()
	qqSend()
}

func qqSend() {
	for {
		qqBot.SendSelfMsg("哈哈")
		time.Sleep(10 * time.Second)
	}
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
