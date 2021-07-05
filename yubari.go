package main

import (
	"flag"
	"time"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/getsentry/raven-go"
	"github.com/go-redis/redis"
	bt "github.com/ikool-cn/gobeanstalk-connection-pool"

	"yubari/bangumi"
	"yubari/pixiv"
	"yubari/telegram"
	"yubari/twitter"
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

	logger = GetLogger("yubari", *flagLogLevel, *flagSyslog)

	cfg := ReadConfig(flagCfgFile)
	logger.Debugf("starting with config: %s", cfg.File)

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
	logger.Debugf("redis connected: %s", redisClient)

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
	es, err := elasticsearch7.NewDefaultClient()
	if err != nil {
		logger.Panic(err)
		return
	}

	telegramBot, err := telegram.NewBot(cfg.Telegram)
	if err != nil {
		logger.Panicf("telegramBot error: %+v", err)
	}
	telegramBot = telegramBot.WithLogger(logger).WithRedis(redisClient).WithQueue(queue).WithES(es)
	telegramBot = telegramBot.WithPixivImg(cfg.Pixiv.ImgPath).WithTwitterImg(cfg.Twitter.ImgPath)

	twitterBot := twitter.NewBot(cfg.Twitter)
	bangumiBot := bangumi.NewBot(cfg.BgmID).WithLogger(logger).WithRedis(redisClient)
	pixivBot := pixiv.NewBot(cfg.Pixiv).WithLogger(logger).WithRedis(redisClient)

	logger.Debugf("bot: telegram: %s", telegramBot.Name)
	go telegramBot.Start()

	logger.Debugf("bot: bangumi: %s", cfg.BgmID)
	bgmUpdate := make(chan string)
	go bangumiBot.StartTrack(60, bgmUpdate)

	logger.Debugf("bot: pixiv: %s", cfg.Pixiv.Username)
	pixivUpdate := make(chan uint64)
	go pixivBot.StartFollow(60, pixivUpdate)

	for {
		select {
		case pID := <-pixivUpdate:
			go telegramBot.SendPixivIllust(telegramBot.SelfID, pID)
		case text := <-bgmUpdate:
			go telegramBot.Send(telegramBot.SelfID, text)
			go twitterBot.Client.Statuses.Update(text, nil)
		}
	}
}
