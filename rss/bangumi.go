package rss

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/everpcpc/chobits/chii"
	"github.com/mmcdole/gofeed"
)

var (
	bangumiProgressEmoji = map[string]string{
		"想看":  "(❛ ◡ ❛)❤",
		"在看":  "(๑• .̫ •๑)",
		"看过":  "(๑>◡<๑)",
		"想读":  "(≧ڡ≦*)",
		"在读":  "(๑•́‧̫•̀๑)",
		"读过":  "♪(๑ᴖ◡ᴖ๑)♪",
		"搁置":  "( •́ﻩ•̀ )",
		"抛弃":  "Σ(-᷅_-᷄๑)",
		"完成了": "(ฅ´ω`ฅ)",
	}
	bgmClient = chii.NewClient(&http.Client{})
)

func getBangumiUpdate(item *gofeed.Item) (output string, err error) {
	r := strings.NewReader(item.Description)
	desc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return
	}
	url, exists := desc.Find("a").Attr("href")
	if !exists {
		err = fmt.Errorf("no url for bangumi update item: %s", item.Title)
		return
	}

	output = item.Title

	tokensURL := strings.Split(url, "/")
	targetType := tokensURL[len(tokensURL)-2]
	switch targetType {
	case "ep", "subject":
		tokensTitle := strings.SplitN(item.Title, " ", 2)
		action := tokensTitle[0]
		emoji := bangumiProgressEmoji[action]
		update := tokensTitle[1]
		var sid int
		sid, err = strconv.Atoi(tokensURL[len(tokensURL)-1])
		if err != nil {
			err = fmt.Errorf("get bangumi id failed with: %s", url)
			return
		}
		var title string
		title, err = getBangumiSubjectTitle(sid, targetType)
		if err != nil {
			err = fmt.Errorf("get subject title failed with %s: %s", url, err)
			return
		}
		if !strings.HasPrefix(update, title) {
			output = emoji + " " + action + "「" + title + "」" + update
		} else {
			ext := strings.TrimSpace(strings.TrimPrefix(update, title))
			output = emoji + " " + action + "「" + title + "」" + ext
		}
	default:
	}
	output += "\n" + url + "\n#Bangumi"
	return
}

func getBangumiSubjectTitle(id int, target string) (string, error) {
	if target == "ep" {
		ep, _, err := bgmClient.Episode.Get(id)
		if err != nil {
			return "", err
		}
		id = ep.SubjectID
	}
	subject, _, err := bgmClient.Subject.Get(id)
	if err != nil {
		return "", err
	}
	return subject.Name, nil
}
