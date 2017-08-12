package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/go-redis/redis"
	"github.com/mmcdole/gofeed"
	"strconv"
	"strings"
	"time"
)

func bgmTrack(id string, ttl int) {
	rssURL := "https://bgm.tv/feed/user/" + id + "/timeline"
	if ttl < 10 {
		ttl = 10
	}
	logger.Debug(rssURL, ttl)
	keyLock := "bgm_" + id + "_lock"
	keyLast := "bgm_" + id + "_last"
	for {
		err := redisClient.Get(keyLock).Err()
		if err == nil {
			time.Sleep(time.Second)
			continue
		} else if err != nil && err != redis.Nil {
			logger.Error("get lock", err)
			time.Sleep(time.Second)
			continue
		}

		if redisClient.Set(keyLock, 0, 10*time.Second).Err() != nil {
			logger.Warning("lock before", err)
		}

		fp := gofeed.NewParser()
		feed, err := fp.ParseURL(rssURL)
		if err != nil {
			logger.Error(err)
			time.Sleep(time.Second)
			continue
		}
		last, err := redisClient.Get(keyLast).Int64()
		if err != nil {
			logger.Warning("get last", err)
			last = 0
		}
		var latest int64
		for _, item := range feed.Items {
			if item.GUID == "" {
				logger.Error("guid not found for", item.Title)
				continue
			}
			tokens := strings.Split(item.GUID, "/")
			guid := tokens[len(tokens)-1]
			id, err := strconv.ParseInt(guid, 10, 64)
			if err != nil {
				logger.Error("guid:", item.GUID)
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
			des := strings.Split(item.Description, `"`)
			if len(des) < 2 {
				logger.Warning("could not get url:", strconv.Quote(item.Description))
				continue
			}
			text := getBangumiUpdate(item.Title, des[1])
			logger.Info(text)
			go twitterBot.Client.Statuses.Update(text, nil)
		}
		if redisClient.Set(keyLast, latest, 0).Err() != nil {
			logger.Error("set last", err)
		}
		if redisClient.Set(keyLock, 0, time.Duration(ttl)*time.Second).Err() != nil {
			logger.Warning("lock after", err)
		}
		time.Sleep(1 * time.Second)
	}
}

func getBangumiUpdate(title, url string) string {
	_title := []rune(title)
	action := string(_title[0:2])
	content := string(_title[2:])
	emoji := emojiBangumi[action]

	subject := getSubjectFromEP(url)
	if subject == "" {
		return emoji + " " + title + " " + url + " #Bangumi"
	}
	return emoji + " " + action + "「" + subject + "」" + content + " " + url + " #Bangumi"
}

func getSubjectFromEP(url string) string {
	tokens := strings.Split(url, "/")
	t := tokens[len(tokens)-2]
	if t != "ep" {
		return ""
	}
	doc, err := goquery.NewDocument(url)
	if err != nil {
		logger.Error(err)
		return ""
	}
	return doc.Find("div#headerSubject h1.nameSingle a").Text()

}
