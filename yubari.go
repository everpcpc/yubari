package main

import (
	"flag"
	"strings"
)

func main() {
	cfgFile := flag.String("c", "config.json", "Config file")
	flagLogger := flag.Int("l", 2, "0 all, 1 std, 2 syslog")
	flagBots := flag.String("b", "qw,tt,tp", "Bots to start: qw qqWatch, tt twitterTrack, tp twitterPics")
	flag.Parse()

	logger = GetLogger(*flagLogger)

	cfg := ReadConfig(cfgFile)
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

	bots := strings.Split(*flagBots, ",")
	botsLaunched := 0
	for _, b := range bots {
		if b == "qw" {
			logger.Notice("Start bot qqWatch")
			messages := make(chan map[string]string)
			go qqBot.Poll(messages)
			go qqWatch(messages)
			botsLaunched++
		} else if b == "tt" {
			logger.Notice("Start bot twitterTrack")
			botsLaunched++
		} else if b == "tp" {
			logger.Notice("Start bot twitterPics")
			botsLaunched++
		} else {
			logger.Warningf("Bot %s is not supported.", b)
		}
	}
	if botsLaunched > 0 {
		select {}
	}
	logger.Notice("Not bots launched.")
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
