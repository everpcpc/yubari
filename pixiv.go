package main

import (
	"strconv"
	"time"

	"github.com/everpcpc/pixiv"
	"github.com/go-redis/redis"
)

var (
	pixivPath = "."
)

func initPixiv(cfg *PixivConfig) error {
	tokenKey := "pixiv:" + cfg.Username + ":auth"
	token := redisClient.HGet(tokenKey, "token").Val()
	refreshToken := redisClient.HGet(tokenKey, "refresh_token").Val()
	tokenDeadline, _ := time.Parse(time.RFC3339, redisClient.HGet(tokenKey, "token_deadline").Val())
	pixiv.HookAuth(func(t, rt string, td time.Time) error {
		v := map[string]interface{}{
			"token":          t,
			"refresh_token":  rt,
			"token_deadline": td.Format(time.RFC3339),
		}
		return redisClient.HMSet(tokenKey, v).Err()
	})
	pixivPath = cfg.ImgPath

	var account *pixiv.Account
	var err error
	if token+refreshToken == "" {
		logger.Debugf("login with %s", cfg.Username)
		account, err = pixiv.Login(cfg.Username, cfg.Password)
	} else {
		logger.Debugf("load auth with %+v", tokenDeadline)
		account, err = pixiv.LoadAuth(token, refreshToken, tokenDeadline)
	}
	if err == nil {
		logger.Debugf("pixiv: %+v", account)
	}
	return err
}

func pixivFollow(cfg *PixivConfig, ttl int) {
	if err := initPixiv(cfg); err != nil {
		logger.Fatal(err)
	}
	if ttl < 10 {
		ttl = 10
	}
	papp := pixiv.NewApp()
	ticker := time.Tick(time.Duration(ttl) * time.Second)
	maxIDKey := "pixiv:" + cfg.Username + ":follow"
	for {
		<-ticker
		maxID, err := redisClient.Get(maxIDKey).Uint64()
		if err != nil && err != redis.Nil {
			logger.Error(err)
			continue
		}

		illusts, _, err := papp.IllustFollow("public", 0)
		if err != nil {
			logger.Error(err)
			continue
		}
		if len(illusts) == 0 {
			continue
		}
		for i := range illusts {
			if maxID >= illusts[i].ID {
				break
			}
			url := pixivURL(illusts[i].ID)
			logger.Infof("post:[%s](%d) %s", illusts[i].User.Name, illusts[i].User.ID, url)
			go telegramBot.send(telegramBot.SelfChatID, url)
		}
		if err := redisClient.Set(maxIDKey, illusts[0].ID, 0).Err(); err != nil {
			logger.Error(err)
		}
	}
}

func pixivURL(id uint64) string {
	return "https://www.pixiv.net/member_illust.php?mode=medium&illust_id=" + strconv.FormatUint(id, 10)
}

func downloadPixiv(id uint64) (int64, error) {
	papp := pixiv.NewApp()
	return papp.Download(id, pixivPath)
}
