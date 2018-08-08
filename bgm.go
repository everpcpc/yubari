package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-redis/redis"
	"github.com/mmcdole/gofeed"
)

func bgmTrack(id string, ttl int) {
	rssURL := "https://bgm.tv/feed/user/" + id + "/timeline"
	if ttl < 10 {
		ttl = 10
	}
	logger.Debugf("%s: %d", rssURL, ttl)
	keyLock := "bgm_lock_" + id
	keyLast := "bgm_last_" + id
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
			go telegramBot.send(telegramBot.SelfChatID, text)
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

func getBangumiUpdate(content, url string) string {
	tokensContent := strings.SplitN(content, " ", 2)

	tokensURL := strings.Split(url, "/")
	t := tokensURL[len(tokensURL)-2]
	switch t {
	case "ep", "subject":
		action := tokensContent[0]
		update := tokensContent[1]
		emoji := emojiBangumi[action]
		title := getSubjectTitleFromURL(url)
		if !strings.HasPrefix(update, title) {
			return emoji + " " + action + "「" + title + "」" + update + " " + url + " #Bangumi"
		}
		ext := strings.TrimSpace(strings.TrimPrefix(update, title))
		return emoji + " " + action + "「" + title + "」" + ext + " " + url + " #Bangumi"
	default:
		return content + " " + url + " #Bangumi"
	}

}

func getSubjectTitleFromURL(url string) string {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		logger.Errorf("%+v", err)
		return ""
	}
	return doc.Find("div#headerSubject h1.nameSingle a").Text()
}
