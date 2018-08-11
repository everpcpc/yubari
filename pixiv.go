package main

import (
	"strconv"
	"time"

	"github.com/everpcpc/pixiv"
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
	maxIDKey := "pixiv:" + cfg.Username + "follow"
	for {
		<-ticker
		illusts, err := papp.IllustFollow("public", 0)
		if err != nil {
			logger.Error(err)
			continue
		}
		maxID, err := redisClient.Get(maxIDKey).Uint64()
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
			go telegramBot.send(telegramBot.SelfChatID, pixivURL(illusts[i].ID))
		}
		if err := redisClient.Set(maxIDKey, illusts[0].ID, 0).Err(); err != nil {
			logger.Error(err)
		}
	}
}

func pixivURL(id uint64) string {
	return "https://www.pixiv.net/member_illust.php?mode=medium&illust_id=" + strconv.FormatUint(id, 10)
}
