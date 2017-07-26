package main

import (
	"flag"
	"fmt"
)

func main() {
	logger = GetLogger(LOGALL)

	cfgfile := flag.String("c", "config.json", "Config file")
	flag.Parse()
	cfg := ReadConfig(cfgfile)
	logger.Infof("Starting with config: %+v", cfg)

	var err error
	qqBot, err = NewQQBot(cfg)
	if err != nil {
		fmt.Println(err)
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
