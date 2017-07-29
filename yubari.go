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
		switch b {
		case "qw":
			logger.Notice("Start bot qqWatch")
			messages := make(chan map[string]string)
			go qqBot.Poll(messages)
			go qqWatch(messages)
			botsLaunched++
		case "tt":
			logger.Notice("Start bot twitterTrack")
			go twitterTrack()
			botsLaunched++
		case "tp":
			logger.Notice("Start bot twitterPics")
			go twitterPics()
			botsLaunched++
		default:
			logger.Warningf("Bot %s is not supported.", b)
		}
	}
	if botsLaunched > 0 {
		select {}
	}
	logger.Notice("Not bots launched.")
}
