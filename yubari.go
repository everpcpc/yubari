package main

import (
	"flag"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/go-redis/redis"
	bt "github.com/ikool-cn/gobeanstalk-connection-pool"

	"github.com/everpcpc/yubari/bangumi"
	"github.com/everpcpc/yubari/pixiv"
	"github.com/everpcpc/yubari/telegram"
	"github.com/everpcpc/yubari/twitter"
)

func NewRedisClient(cfg *RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	_, err := client.Ping().Result()
	// Output: PONG <nil>
	return client, err
}

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

	queue := &bt.Pool{
		Dial: func() (*bt.Conn, error) {
			return bt.Dial(cfg.BeanstalkAddr)
		},
		MaxIdle:     10,
		MaxActive:   100,
		IdleTimeout: 60 * time.Second,
		MaxLifetime: 180 * time.Second,
		Wait:        true,
	}

	telegramBot, err := telegram.NewBot(cfg.Telegram)
	if err != nil {
		logger.Panicf("TelegramBot error: %+v", err)
	}
	telegramBot = telegramBot.WithLogger(logger).WithRedis(redisClient).WithQueue(queue).WithPixivImg(cfg.Pixiv.ImgPath).WithTwitterImg(cfg.Twitter.ImgPath)

	twitterBot := twitter.NewBot(cfg.Twitter)
	bangumiBot := bangumi.NewBot(cfg.BgmID).WithLogger(logger).WithRedis(redisClient)
	pixivBot := pixiv.NewBot(cfg.Pixiv).WithLogger(logger).WithRedis(redisClient)

	logger.Debugf("Bot: telegram: %+v", telegramBot)
	go telegramBot.Start()

	logger.Debugf("Bot: bangumi: %+v", bangumiBot)
	bgmUpdate := make(chan string)
	go bangumiBot.StartTrack(10, bgmUpdate)

	logger.Debugf("Bot: pixiv: %+v", pixivBot)
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
