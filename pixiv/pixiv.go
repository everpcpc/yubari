package pixiv

import (
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/everpcpc/pixiv"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Username string `json:"username"`
	Password string `json:"password"`
	ImgPath  string `json:"imgPath"`
}

type Bot struct {
	redis  *redis.Client
	logger *logrus.Logger
	config *Config
}

func NewBot(cfg *Config) *Bot {
	b := &Bot{config: cfg}
	return b
}

func (b *Bot) WithLogger(logger *logrus.Logger) *Bot {
	b.logger = logger
	return b
}

func (b *Bot) WithRedis(rds *redis.Client) *Bot {
	b.redis = rds
	return b
}

func (b *Bot) init() error {
	tokenKey := "pixiv:" + b.config.Username + ":auth"
	token := b.redis.HGet(tokenKey, "token").Val()
	refreshToken := b.redis.HGet(tokenKey, "refresh_token").Val()
	tokenDeadline, _ := time.Parse(time.RFC3339, b.redis.HGet(tokenKey, "token_deadline").Val())
	pixiv.HookAuth(func(t, rt string, td time.Time) error {
		v := map[string]interface{}{
			"token":          t,
			"refresh_token":  rt,
			"token_deadline": td.Format(time.RFC3339),
		}
		return b.redis.HMSet(tokenKey, v).Err()
	})

	var account *pixiv.Account
	var err error
	if token+refreshToken == "" {
		b.logger.Debugf("pixiv login with %s", b.config.Username)
		account, err = pixiv.Login(b.config.Username, b.config.Password)
	} else {
		b.logger.Debugf("pixiv auth loaded with %+v", tokenDeadline)
		account, err = pixiv.LoadAuth(token, refreshToken, tokenDeadline)
	}
	if err == nil {
		b.logger.Debugf("pixiv started for %+v", account)
	}
	return err
}

func (b *Bot) StartFollow(ttl int, output chan uint64) {
	if err := b.init(); err != nil {
		b.logger.Fatal(err)
	}
	if ttl < 10 {
		ttl = 10
	}
	papp := pixiv.NewApp()
	ticker := time.NewTicker(time.Duration(ttl) * time.Second)
	maxIDKey := "pixiv:" + b.config.Username + ":follow"
	for {
		<-ticker.C
		b.logger.Debugf("fetching %s", maxIDKey)
		maxID, err := b.redis.Get(maxIDKey).Uint64()
		if err != nil && err != redis.Nil {
			b.logger.Error(err)
			continue
		}

		illusts, _, err := papp.IllustFollow("public", 0)
		if err != nil {
			b.logger.Error(err)
			continue
		}
		if len(illusts) == 0 {
			continue
		}
		for i := range illusts {
			if maxID >= illusts[i].ID {
				break
			}
			b.logger.Infof("pixiv post:[%s](%d) %s", illusts[i].User.Name, illusts[i].User.ID, URLWithID(illusts[i].ID))
			output <- illusts[i].ID
		}
		if err := b.redis.Set(maxIDKey, illusts[0].ID, 0).Err(); err != nil {
			b.logger.Error(err)
		}
	}
}

func URLWithID(id uint64) string {
	return "https://www.pixiv.net/artworks/" + strconv.FormatUint(id, 10)
}

func ParseURL(s string) uint64 {
	urlPattern := regexp.MustCompile(`https:\/\/www\.pixiv\.net\/member_illust\.php\?(illust_id=\d+|mode=medium|\&)+`)
	matchURL := urlPattern.FindString(s)
	u, err := url.Parse(matchURL)
	if err != nil {
		return 0
	}
	r, err := strconv.ParseUint(u.Query().Get("illust_id"), 10, 0)
	if err != nil {
		return 0
	}
	return r
}

func Download(id uint64, dir string) ([]int64, error) {
	papp := pixiv.NewApp().WithDownloadTimeout(time.Minute)
	return papp.Download(id, dir)
}
