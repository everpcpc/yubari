package main

import (
	"flag"
	"strings"
)

func main() {
	flagCfgFile := flag.String("config", "conf/config.json", "Config file")
	flagSyslog := flag.Bool("syslog", false, "also log to syslog")
	flagLogLevel := flag.String("loglevel", "debug", "debug, info, notice, warning, error")
	flagBots := flag.String(
		"bots", "tt,ts,tg,bgm",
		"Bots to start: qw qqWatch, tt twitterTrack, ts twitterSelf, tg telegram, bgm bgmTrack")
	flag.Parse()

	var logFlags byte
	if *flagSyslog {
		logFlags = logFlags | LOGSYS
	}
	logger = GetLogger("yubari", *flagLogLevel, logFlags|LOGCOLOR)

	cfg := ReadConfig(flagCfgFile)
	logger.Debugf("Starting with config: %+v", cfg)

	var err error
	redisClient, err = NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Panic(err)
		return
	}
	defer redisClient.Close()
	logger.Debugf("Redis connected: %+v", redisClient)

	// qqBot = NewQQBot(cfg)
	// defer qqBot.Client.Close()
	// logger.Debugf("QQBot: %+v", qqBot)

	twitterBot = NewTwitterBot(cfg.Twitter)
	logger.Debugf("TwitterBot: %+v", twitterBot)

	telegramBot = NewTelegramBot(cfg.Telegram, cfg.BeanstalkAddr)
	logger.Debugf("TelegramBot: %+v", telegramBot)

	bots := strings.Split(*flagBots, ",")
	botsLaunched := 0
	for _, b := range bots {
		switch b {
		// case "qw":
		// logger.Debug("Bot: qqWatch")
		// messages := make(chan map[string]string)
		// go qqBot.Poll(messages)
		// go qqWatch(messages)
		// botsLaunched++
		case "tt":
			logger.Debug("Bot: twitterTrack")
			go twitterBot.Track()
			botsLaunched++
		case "ts":
			logger.Debug("Bot: twitterSelf")
			go twitterBot.Self()
			botsLaunched++
		case "tg":
			logger.Debug("Bot: telegram")
			go telegramBot.tgBot()
			go telegramBot.delMessage()
			botsLaunched++
		case "bgm":
			logger.Debug("Bot: bgmTrack")
			go bgmTrack(cfg.BgmID, 10)
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
