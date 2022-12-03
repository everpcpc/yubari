package pixiv

import (
	"math/rand"
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

var (
	proxy *url.URL
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

	papp := pixiv.NewApp().WithDownloadTimeout(2 * time.Minute).WithTmpdir(cfg.TmpDir)
	if cfg.Proxy != "" {
		if u, err := url.Parse(cfg.Proxy); err != nil {
			return nil, err
		} else {
			papp = papp.WithDownloadProxy(u)
		}
	}
	b.papp = papp
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
	if b.config.Proxy != "" {
		if u, err := url.Parse(b.config.Proxy); err != nil {
			return err
		} else {
			proxy = u
		}
	}

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

func (b *Bot) Download(id uint64) ([]int64, error) {
	// illust, err := b.papp.IllustDetail(id)
	// if err != nil {
	// 	err = errors.Wrapf(err, "illust %d detail error", id)
	// 	return
	// }
	// if illust == nil {
	// 	err = errors.Wrapf(err, "illust %d is nil", id)
	// 	return
	// }
	// if illust.MetaSinglePage == nil {
	// 	err = errors.Wrapf(err, "illust %d has no single page", id)
	// 	return
	// }

	// var urls []string
	// if illust.MetaSinglePage.OriginalImageURL == "" {
	// 	for _, img := range illust.MetaPages {
	// 		urls = append(urls, img.Images.Original)
	// 	}
	// } else {
	// 	urls = append(urls, illust.MetaSinglePage.OriginalImageURL)
	// }

	// dclient := &http.Client{}
	// if a.proxy != nil {
	// 	dclient.Transport = &http.Transport{
	// 		Proxy: http.ProxyURL(a.proxy),
	// 	}
	// }
	// if a.timeout != 0 {
	// 	dclient.Timeout = a.timeout
	// }

	// for _, u := range urls {
	// 	size, e := download(dclient, u, path, filepath.Base(u), a.tmpDir, false)
	// 	if e != nil {
	// 		err = errors.Wrapf(e, "download url %s failed", u)
	// 		return
	// 	}
	// 	sizes = append(sizes, size)
	// }

	// return

	return b.papp.Download(id, b.config.ImgPath)
}

func (b *Bot) RandomPic() (filePath string, fileName string, err error) {
	files, err := filepath.Glob(filepath.Join(b.config.ImgPath, "*"))
	if err != nil {
		err = errors.Wrap(err, "glob")
		return
	}
	if files == nil {
		err = errors.New("no files")
		return
	}
	rand.Seed(time.Now().UnixNano())
	filePath = files[rand.Intn(len(files))]
	fileName = filepath.Base(filePath)
	return
}

func (b *Bot) Probate(_id string) error {
	return os.Rename(
		filepath.Join(b.config.ImgPath, _id),
		filepath.Join(b.config.ImgPath, "probation", _id),
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
