package meili

import "encoding/json"

type Config struct {
	Host   string `json:"host"`
	APIKey string `json:"apiKey"`
}

type Article struct {
	ID      int64  `json:"id"`
	Date    int64  `json:"date"`
	User    int64  `json:"user"`
	Content string `json:"content"`
}

func DecodeArticles(hits []interface{}) (articles []Article, err error) {
	articles = []Article{}
	tmp, err := json.Marshal(hits)
	if err != nil {
		return
	}
	err = json.Unmarshal(tmp, &articles)
	if err != nil {
		return
	}
	return
}
