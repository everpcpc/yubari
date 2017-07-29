package main

import (
	"flag"
	"strings"
)

func main() {
	cfgfile := flag.String("c", "config.json", "Config file")
	logto := flag.Int("l", 2, "0 all, 1 std, 2 syslog")
	bots := flag.String("b", "qw,tt,tp", "Bots to start: qw qqWatch, tt twitterTrack, tp twitterPics")
	flag.Parse()

	logger = GetLogger(*logto)

	cfg := ReadConfig(cfgfile)
	logger.Infof("Starting with config: %+v", cfg)

	var err error
	rds, err = NewRedisClient(cfg.RedisCfg)
	if err != nil {
		logger.Panic(err)
		return
	}
	defer rds.Close()
	logger.Infof("Redis connected: %+v", rds)

	qqBot = NewQQBot(cfg)
	defer qqBot.Client.Close()
	logger.Infof("QQBot: %+v", qqBot)

	bs := strings.Split(*bots, ",")
	for _, b := range bs {
		if b == "qw" {
			messages := make(chan map[string]string)
			go qqBot.Poll(messages)
			go qqWatch(messages)
		}
	}
	select {}
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
			go qqBot.NoticeMention(msg["msg"], msg["group"])
			go qqBot.CheckRepeat(msg["msg"], msg["group"])
			logger.Infof("(%s)[%s]:{%s}", msg["group"], msg["qq"], msg["msg"])
		default:
			logger.Info(msg)
		}
	}
}
