package main

import (
	"errors"
	"github.com/go-redis/redis"
	"github.com/mmcdole/gofeed/rss"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parseRssURL(url string) (*rss.Feed, error) {
	fp := rss.Parser{}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		defer func() {
			ce := resp.Body.Close()
			if ce != nil {
				err = ce
			}
		}()
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(resp.Status)
	}

	return fp.Parse(resp.Body)
}

func bgmTrack(id string, ttl int) {
	rssURL := "https://bgm.tv/feed/user/" + id + "/timeline"
	if ttl < 10 {
		ttl = 10
	}
	logger.Debug(rssURL, ttl)
	keyLock := "bgm_" + id + "_lock"
	keyLast := "bgm_" + id + "_last"
	i := 1
	for {
		err := redisClient.Get(keyLock).Err()
		if err == nil {
			i = 1
			continue
		} else if err != nil && err != redis.Nil {
			logger.Error("get lock", err)
			time.Sleep(time.Duration(i) * time.Second)
			i += int(i/2.0) + 1
			continue
		}

		if redisClient.Set(keyLock, 0, 10*time.Second).Err() != nil {
			logger.Error("set lock first", err)
		}

		rss, err := parseRssURL(rssURL)
		if err != nil {
			logger.Error(err)
			time.Sleep(time.Duration(i) * time.Second)
			i += int(i/2.0) + 1
			continue
		}
		last, err := redisClient.Get(keyLast).Int64()
		if err != nil {
			logger.Error("get last", err)
			last = 0
		}
		var latest int64
		for _, item := range rss.Items {
			if item.GUID == nil {
				continue
			}
			tokens := strings.Split(item.GUID.Value, "/")
			guid := tokens[len(tokens)-1]
			id, err := strconv.ParseInt(guid, 10, 64)
			if err != nil {
				logger.Error("guid:", item.GUID.Value)
				continue
			}
			if id > latest {
				latest = id
			}
			if last == 0 {
				last = id
				break
			}
			if id <= last {
				break
			}
			emoji := emojiBangumi[string([]rune(item.Title)[0:2])]
			des := strings.Split(item.Description, `"`)
			var desURL string
			if len(des) > 2 {
				desURL = des[1]
			}
			text := emoji + " " + item.Title + " " + desURL + " #Bangumi"
			logger.Info(text)
			go twitterBot.Client.Statuses.Update(text, nil)
		}
		if redisClient.Set(keyLast, latest, 0).Err() != nil {
			logger.Error("set last", err)
		}
		if redisClient.Set(keyLock, 0, time.Duration(ttl)*time.Second).Err() != nil {
			logger.Error("set lock", err)
		}
		time.Sleep(1 * time.Second)
	}
}
