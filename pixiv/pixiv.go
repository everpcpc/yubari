package pixiv

import (
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/everpcpc/pixiv"
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Username string `json:"username"`
	Password string `json:"password"`
	ImgPath  string `json:"imgPath"`
	TmpDir   string `json:"tmpDir"`
	Proxy    string `json:"proxy"`
}

type Bot struct {
	redis  *redis.Client
	logger *logrus.Logger
	config *Config
	papp   *pixiv.AppPixivAPI
}

func NewBot(cfg *Config, redisClient *redis.Client, logger *logrus.Logger) (*Bot, error) {
	b := &Bot{
		redis:  redisClient,
		logger: logger,
		config: cfg,
	}

	if err := b.init(); err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	downloadClient := &http.Client{
		Timeout: time.Minute,
	}
	if cfg.Proxy != "" {
		u, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, err
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(u),
		}
		downloadClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(u),
		}
	}

	b.papp = pixiv.NewApp().WithTmpdir(cfg.TmpDir)
	return b, nil
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
		b.logger.Debugf("pixiv auth loaded with %s", tokenDeadline)
		account, err = pixiv.LoadAuth(token, refreshToken, tokenDeadline)
	}
	if err == nil {
		b.logger.Debugf("pixiv started for %+v", account)
	}
	return err
}

func (b *Bot) StartFollow(ttl int, output chan uint64) {
	if ttl < 10 {
		ttl = 10
	}
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

		illusts, _, err := b.papp.IllustFollow("public", 0)
		if err != nil {
			b.logger.Error(err)
			continue
		}
		if len(illusts) == 0 {
			continue
		}
		for i := range illusts {
			illust := illusts[i]
			if maxID >= illust.ID {
				break
			}
			if illust.IllustAIType == pixiv.IllustAITypeAIGenerated {
				b.logger.Infof("pixiv post:[%s](%d) (AI Generated)", illust.User.Name, illust.User.ID)
				continue
			}
			b.logger.Infof("pixiv post:[%s](%d) %s", illust.User.Name, illust.User.ID, URLWithID(illust.ID))
			output <- illust.ID
		}
		if err := b.redis.Set(maxIDKey, illusts[0].ID, 0).Err(); err != nil {
			b.logger.Error(err)
		}
	}
}

func (b *Bot) Download(id uint64) ([]int64, error) {
	fn := func(illust *pixiv.Illust) string {
		subdir := illust.CreateDate.Format("2006-01")
		path := filepath.Join(b.config.ImgPath, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			b.logger.Warn(err)
			return b.config.ImgPath
		}
		return path
	}
	return b.papp.Download(id, fn)
}

func (b *Bot) RandomPic() (filePath string, err error) {
	now := time.Now()
	current := now.Format("2006-01")
	previous := now.AddDate(0, -1, 0).Format("2006-01")
	currentFiles, err := filepath.Glob(filepath.Join(b.config.ImgPath, current, "*"))
	if err != nil {
		err = errors.Wrap(err, "glob current")
		return
	}
	previousFiles, err := filepath.Glob(filepath.Join(b.config.ImgPath, previous, "*"))
	if err != nil {
		err = errors.Wrap(err, "glob previous")
		return
	}
	files := append(currentFiles, previousFiles...)
	if files == nil {
		err = errors.New("no files")
		return
	}
	rand.Seed(time.Now().UnixNano())
	filePath = files[rand.Intn(len(files))]
	return
}

func (b *Bot) Probate(target string) error {
	return os.Rename(
		filepath.Join(b.config.ImgPath, target),
		filepath.Join(b.config.ImgPath, "probation", filepath.Base(target)),
	)
}

func URLWithID(id uint64) string {
	return "https://www.pixiv.net/artworks/" + strconv.FormatUint(id, 10)
}

func ParseURL(s string) uint64 {
	urlPattern := regexp.MustCompile(`https:\/\/www\.pixiv\.net\/artworks\/(\d+)`)
	matches := urlPattern.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0
	}
	r, err := strconv.ParseUint(matches[1], 10, 0)
	if err != nil {
		return 0
	}
	return r
}
