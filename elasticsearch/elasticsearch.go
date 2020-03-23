package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
)

var (
	mapping = `{
		"mappings": {
			"properties":{
				"content": {
					"type": "text",
					"analyzer": "ik_max_word",
					"search_analyzer": "ik_smart"
				},
				"message_id": {
					"type": "long"
				},
				"date": {
					"type": "date"
				}
			}
		}
	}`
	query = map[string]interface{}{
		"highlight": map[string]interface{}{
			"pre_tags":  []string{"<b>"},
			"post_tags": []string{"</b>"},
			"fields": map[string]interface{}{
				"content": map[string]interface{}{
					"fragment_size":       15,
					"number_of_fragments": 3,
					"fragmenter":          "span",
				},
			},
		},
	}
)

type Article struct {
	Content   string `json:"content"`
	MessageID int64  `json:"message_id"`
	Date      int64  `json:"date"`
}

type SearchResponse struct {
	Took int64
	Hits struct {
		Total struct {
			Value int64
		}
		Hits []*SearchHit
	}
}

type SearchHit struct {
	Score   float64 `json:"_score"`
	Index   string  `json:"_index"`
	Type    string  `json:"_type"`
	Version int64   `json:"_version,omitempty"`

	Source Article `json:"_source"`

	Highlight struct {
		Content []string
	}
}

func checkIndexExist(es *elasticsearch7.Client, idx string) (bool, error) {
	res, err := es.Indices.Exists([]string{idx})

	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return false, nil
	}
	return true, nil
}

func createIndex(es *elasticsearch7.Client, idx string) error {
	res, err := es.Indices.Create(
		idx,
		es.Indices.Create.WithBody(strings.NewReader(mapping)),
		es.Indices.Create.WithWaitForActiveShards("1"),
	)
	if err != nil {
		return fmt.Errorf("create index %s error: %+v", idx, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("create index %s error: %+v", idx, res)
	}
	return nil
}

func storeMessage(es *elasticsearch7.Client, idx string, message *Article) error {
	ret, err := json.Marshal(message)
	if err != nil {
		return err
	}
	res, err := es.Index(
		idx,
		bytes.NewReader(ret),
		es.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("store to es error: %+v", err)
	}

	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("store message %s error: %+v", idx, res)
	}
	return nil
}

func searchMessage(es *elasticsearch7.Client, idx, q string, from int) (r *SearchResponse, err error) {
	res, err := es.Search(
		es.Search.WithContext(context.TODO()),
		es.Search.WithIndex(idx),
		es.Search.WithDf("content"),
		es.Search.WithBody(esutil.NewJSONReader(&query)),
		es.Search.WithTrackTotalHits(true),
		es.Search.WithQuery(q),
		es.Search.WithSize(5),
		es.Search.WithFrom(from),
		es.Search.WithPretty(),
	)
	if err != nil {
		err = fmt.Errorf("Getting response error: %s", err)
		return
	}
	defer res.Body.Close()
	if res.IsError() {
		err = fmt.Errorf("Search error: %s", err)
		return
	}

	if err = json.NewDecoder(res.Body).Decode(&r); err != nil {
		err = fmt.Errorf("Decoding response error: %+v", err)
	}

	return
}
