package telegram

import (
	"fmt"
	"strings"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
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
)

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
