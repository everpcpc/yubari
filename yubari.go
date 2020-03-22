package main

import (
	"flag"
	"strings"

	"github.com/getsentry/raven-go"

	"github.com/everpcpc/yubari/bangumi"
	"github.com/everpcpc/yubari/pixiv"
	"github.com/everpcpc/yubari/telegram"
)

func main() {
	flagCfgFile := flag.String("config", "conf/config.json", "Config file")
	flagSyslog := flag.Bool("syslog", false, "also log to syslog")
	flagLogLevel := flag.String("loglevel", "debug", "debug, info, notice, warning, error")
	flagBots := flag.String(
		"bots", "tg,bgm,pixiv",
		`Bots to start:
			tg telegram,
			bgm bgm Track,
			pixiv pixiv Follow`)
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

	var err error
	redisClient, err = NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Panic(err)
		return
	}
	defer redisClient.Close()
	logger.Debugf("Redis connected: %+v", redisClient)

	bangumiBot := bangumi.NewBot(cfg.BgmID).WithLogger(logger).WithRedis(redisClient)

	telegramBot, err := telegram.NewBot(cfg.Telegram)
	if err != nil {
		logger.Panicf("TelegramBot error: %+v", err)
	}
	telegramBot = telegramBot.WithLogger(logger).WithRedis(redisClient).WithBeanstalkd(cfg.BeanstalkAddr).WithPixivImg(cfg.Pixiv.ImgPath).WithTwitterImg(cfg.Twitter.ImgPath)
	logger.Debugf("TelegramBot: %+v", telegramBot)

	pixivBot := pixiv.NewBot(cfg.Pixiv).WithLogger(logger).WithRedis(redisClient)

	bots := strings.Split(*flagBots, ",")
	botsLaunched := 0
	for _, b := range bots {
		switch b {
		case "tg":
			logger.Debug("Bot: telegram")
			go telegramBot.Start()
			botsLaunched++
		case "bgm":
			logger.Debug("Bot: bgmTrack")
			go bangumiBot.StartTrack(10)
			botsLaunched++
		case "pixiv":
			logger.Debug("Bot: pixivFollow")
			pixivOutput := make(chan uint64)
			go pixivBot.StartFollow(20, pixivOutput)
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
