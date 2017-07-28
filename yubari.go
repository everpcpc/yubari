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
	defer rds.Close()
	logger.Infof("Redis connected: %+v", rds)

	qqBot = NewQQBot(cfg)
	defer qqBot.Pool.Close()
	logger.Infof("QQBot: %+v", qqBot)

	messages := make(chan map[string]string)
	go qqBot.Poll(messages)
	go qqWatch(messages)
}

func qqWatch(messages chan map[string]string) {
	ignoreMap := make(map[string]struct{})
	for _, q := range qqBot.Cfg.QQIgnore {
		ignoreMap[q] = struct{}{}
	}

	for msg := range messages {
		switch msg["event"] {
		case "PrivateMsg":
			logger.Infof("[%s]:{%s}", msg["qq"], msg["msg"])
		case "GroupMsg":
			if _, ok := ignoreMap[msg["qq"]]; ok {
				logger.Debugf("Ignore (%s)[%s]:{%s}", msg["group"], msg["qq"], msg["msg"])
				continue
			}
			logger.Infof("(%s)[%s]:{%s}", msg["group"], msg["qq"], msg["msg"])
		default:
			logger.Info(msg)
		}
	}
}
