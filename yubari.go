package main

import (
	"flag"
	"time"

	sentry "github.com/getsentry/sentry-go"
	"github.com/go-redis/redis"
	bt "github.com/ikool-cn/gobeanstalk-connection-pool"
	meilisearch "github.com/meilisearch/meilisearch-go"
	"gopkg.in/gographics/imagick.v3/imagick"

	"yubari/mastodon"
	"yubari/pixiv"
	"yubari/rss"
	"yubari/telegram"
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

	imagick.Initialize()
	defer imagick.Terminate()

	logger = GetLogger("yubari", *flagLogLevel, *flagSyslog)

	cfg := ReadConfig(flagCfgFile)
	logger.Debugf("starting with config: %s", cfg.File)

	if cfg.SentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn: cfg.SentryDSN,
		})
		if err != nil {
			logger.Fatalf("Sentry initialization failed: %s", err)
		}
	}

	redisClient, err := NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal(err)
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

	meili := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   cfg.Meilisearch.Host,
		APIKey: cfg.Meilisearch.APIKey,
	})

	mastodonBot := mastodon.NewBot(cfg.Mastodon)
	pixivBot, err := pixiv.NewBot(cfg.Pixiv, redisClient, logger)
	if err != nil {
		logger.Fatalf("pixivBot error: %s", err)
	}

	telegramBot, err := telegram.NewBot(cfg.Telegram)
	if err != nil {
		logger.Fatalf("telegramBot error: %s", err)
	}
	telegramBot = telegramBot.WithLogger(logger).WithRedis(redisClient).WithQueue(queue).WithMeilisearch(meili)
	telegramBot = telegramBot.WithPixiv(pixivBot)
	if cfg.Telegram.OpenAI != nil {
		logger.Debug("bot: openai: enabled")
		telegramBot = telegramBot.WithOpenAI(cfg.Telegram.OpenAI)
	}

	rssUpdate := make(chan string)
	logger.Debugf("bot: rss: %+v", cfg.RSS.Feeds)
	rss.NewBot(cfg.RSS, rssUpdate).WithLogger(logger).WithRedis(redisClient).Start()

	logger.Debugf("bot: telegram: %s", telegramBot.Name)
	go telegramBot.Start()

	logger.Debugf("bot: pixiv: %s", cfg.Pixiv.Username)
	pixivUpdate := make(chan uint64)
	go pixivBot.StartFollow(300, pixivUpdate)

	for {
		select {
		case pID := <-pixivUpdate:
			go telegramBot.SendPixivCandidate(telegramBot.AdmissionID, pID)
		case text := <-rssUpdate:
			go telegramBot.Send(telegramBot.SelfID, text)
			go mastodonBot.NewStatus(text, true)
		}
	}
}
