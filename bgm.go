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
	logger.Debugf("%s: %d", rssURL, ttl)
	keyLock := "bgm_" + id + "_lock"
	keyLast := "bgm_" + id + "_last"
	for {
		err := redisClient.Get(keyLock).Err()
		if err == nil {
			time.Sleep(time.Second)
			continue
		} else if err != nil && err != redis.Nil {
			logger.Errorf("get lock %+v", err)
			time.Sleep(time.Second)
			continue
		}

		if redisClient.Set(keyLock, 0, 10*time.Second).Err() != nil {
			logger.Warningf("lock before %+v", err)
		}

		fp := gofeed.NewParser()
		feed, err := fp.ParseURL(rssURL)
		if err != nil {
			logger.Errorf("%+v", err)
			time.Sleep(time.Second)
			continue
		}
		last, err := redisClient.Get(keyLast).Int64()
		if err != nil {
			logger.Warningf("get last %+v", err)
			last = 0
		}
		var latest int64
		for _, item := range feed.Items {
			if item.GUID == "" {
				logger.Errorf("guid not found for %+v", item.Title)
				continue
			}
			tokens := strings.Split(item.GUID, "/")
			guid := tokens[len(tokens)-1]
			id, err := strconv.ParseInt(guid, 10, 64)
			if err != nil {
				logger.Errorf("guid: %+v", item.GUID)
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
				logger.Warningf("could not get url: %+v", strconv.Quote(item.Description))
				continue
			}
			text := getBangumiUpdate(item.Title, des[1])
			logger.Info(text)
			go twitterBot.Client.Statuses.Update(text, nil)
		}
		if redisClient.Set(keyLast, latest, 0).Err() != nil {
			logger.Errorf("set last %+v", err)
		}
		if redisClient.Set(keyLock, 0, time.Duration(ttl)*time.Second).Err() != nil {
			logger.Warningf("lock after %+v", err)
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
		logger.Errorf("%+v", err)
		return ""
	}
	return doc.Find("div#headerSubject h1.nameSingle a").Text()

}
