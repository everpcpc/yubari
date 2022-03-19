package rss

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

func getDoubanUpdate(item *gofeed.Item) (output string, err error) {
	r := strings.NewReader(item.Description)
	desc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return
	}
	url, exists := desc.Find("a").Attr("href")
	if !exists {
		err = fmt.Errorf("no url for douban update item: %s", item.Title)
		return
	}
	title, exists := desc.Find("a").Attr("title")
	if !exists {
		err = fmt.Errorf("no title for douban update item: %s", item.Title)
		return
	}
	output = item.Title + "「" + title + "」" + "\n" + url + "\n#Douban"
	return
}
