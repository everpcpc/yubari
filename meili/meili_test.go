package meili

import (
	"fmt"
	"testing"
	"time"

	meilisearch "github.com/meilisearch/meilisearch-go"
	"github.com/stretchr/testify/require"
)

func TestStoreAndSearch(t *testing.T) {
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host: "http://localhost:7700",
	})
	health, err := client.Health()
	fmt.Println(health)
	require.Nil(t, err)

	idx := client.Index(fmt.Sprintf("test-%d", time.Now().UnixNano()))

	message := Article{
		ID:      987654321,
		User:    1234,
		Date:    123456789,
		Content: "看需求……\nns有主机和掌机模式\nlite是阉割轻量版，只有掌机模式",
	}
	taskInfo, err := idx.AddDocuments(&message, "id")
	require.Nil(t, err)

loop1:
	for {
		time.Sleep(time.Second)
		task, err := client.GetTask(taskInfo.TaskUID)
		require.Nil(t, err)
		switch task.Status {
		case meilisearch.TaskStatusSucceeded:
			break loop1
		case meilisearch.TaskStatusFailed:
			panic("meili add task failed")
		default:
			fmt.Printf("waiting for add document task, current: %v\n", task.Status)
		}
	}

	res, err := idx.Search("主机", &meilisearch.SearchRequest{
		Limit: 5,
	})
	require.Nil(t, err)
	require.Equal(t, int64(1), res.EstimatedTotalHits)

	hits, err := DecodeArticles(res.Hits)
	require.Nil(t, err)
	require.Equal(t, message.Content, hits[0].Content)
	require.Equal(t, message.User, hits[0].User)
	require.Equal(t, message.Date, hits[0].Date)

	// test replace
	message2 := Article{
		ID:      987654321,
		User:    4567,
		Date:    13567890124,
		Content: "lite是阉割轻量版，只有掌机模式",
	}
	taskInfo, err = idx.AddDocuments(&message2, "id")
	require.Nil(t, err)

loop2:
	for {
		time.Sleep(time.Second)
		task, err := client.GetTask(taskInfo.TaskUID)
		require.Nil(t, err)
		switch task.Status {
		case meilisearch.TaskStatusSucceeded:
			break loop2
		case meilisearch.TaskStatusFailed:
			panic("meili add task failed")
		default:
			fmt.Printf("waiting for add document task, current: %v\n", task.Status)
		}
	}
	res, err = idx.Search("模式", &meilisearch.SearchRequest{
		Limit: 5,
	})
	require.Nil(t, err)
	require.Equal(t, int64(1), res.EstimatedTotalHits)

	hits, err = DecodeArticles(res.Hits)
	require.Nil(t, err)
	require.Equal(t, message2.Content, hits[0].Content)
	require.Equal(t, message2.User, hits[0].User)
	require.Equal(t, message2.Date, hits[0].Date)

}
