package main

import (
	"flag"

	"github.com/getsentry/raven-go"

	"github.com/everpcpc/yubari/bangumi"
	"github.com/everpcpc/yubari/pixiv"
	"github.com/everpcpc/yubari/telegram"
	"github.com/everpcpc/yubari/twitter"
)

func main() {
	flagCfgFile := flag.String("config", "conf/config.json", "Config file")
	flagSyslog := flag.Bool("syslog", false, "also log to syslog")
	flagLogLevel := flag.String("loglevel", "debug", "debug, info, notice, warning, error")
	flag.Parse()

	var logFlags byte
	if *flagSyslog {
		logFlags = logFlags | LOGSYS
	}
	logger = GetLogger("yubari", *flagLogLevel, logFlags|LOGCOLOR)

	cfg := ReadConfig(flagCfgFile)
	logger.Debugf("Starting with config: %+v", cfg)

	if cfg.SentryDSN != "" {
		raven.SetDSN(cfg.SentryDSN)
		raven.CapturePanic(func() {
			// do all of the scary things here
		}, nil)
	}

	redisClient, err := NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Panic(err)
		return
	}
	defer redisClient.Close()
	logger.Debugf("Redis connected: %+v", redisClient)

	telegramBot, err := telegram.NewBot(cfg.Telegram)
	if err != nil {
		logger.Panicf("TelegramBot error: %+v", err)
	}
	telegramBot = telegramBot.WithLogger(logger).WithRedis(redisClient).WithBeanstalkd(cfg.BeanstalkAddr).WithPixivImg(cfg.Pixiv.ImgPath).WithTwitterImg(cfg.Twitter.ImgPath)
	logger.Debugf("TelegramBot: %+v", telegramBot)

	twitterBot := twitter.NewBot(cfg.Twitter)
	bangumiBot := bangumi.NewBot(cfg.BgmID).WithLogger(logger).WithRedis(redisClient)
	pixivBot := pixiv.NewBot(cfg.Pixiv).WithLogger(logger).WithRedis(redisClient)

	logger.Debug("Bot: telegram")
	go telegramBot.Start()

	logger.Debug("Bot: bangumi")
	bgmUpdate := make(chan string)
	go bangumiBot.StartTrack(10, bgmUpdate)

	logger.Debug("Bot: pixiv")
	pixivUpdate := make(chan uint64)
	go pixivBot.StartFollow(20, pixivUpdate)

	select {
	case pID := <-pixivUpdate:
		go telegramBot.SendPixivIllust(telegramBot.SelfID, pID)
	case text := <-bgmUpdate:
		go telegramBot.Send(telegramBot.SelfID, text)
		go twitterBot.Client.Statuses.Update(text, nil)
	}
}
