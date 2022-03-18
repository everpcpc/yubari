package rss

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
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

	tokensURL := strings.Split(url, "/")
	targetType := tokensURL[len(tokensURL)-2]
	switch targetType {
	case "ep", "subject":
		tokensTitle := strings.SplitN(item.Title, " ", 2)
		action := tokensTitle[0]
		emoji := bangumiProgressEmoji[action]
		update := tokensTitle[1]
		title, _ := getBangumiSubjectTitleFromURL(url)
		if !strings.HasPrefix(update, title) {
			output = emoji + " " + action + "「" + title + "」" + update
		} else {
			ext := strings.TrimSpace(strings.TrimPrefix(update, title))
			output = emoji + " " + action + "「" + title + "」" + ext
		}
	default:
		output = item.Title
	}
	output += " #Bangumi " + url
	return
}

func getBangumiSubjectTitleFromURL(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", err
	}
	return doc.Find("div#headerSubject h1.nameSingle a").Text(), nil
}
