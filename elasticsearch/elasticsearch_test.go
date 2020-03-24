package elasticsearch

import (
	"fmt"
	"testing"
	"time"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/stretchr/testify/require"
)

func TestStoreAndSearch(t *testing.T) {
	es, err := elasticsearch7.NewDefaultClient()
	require.Nil(t, err)

	idx := fmt.Sprintf("test-%d", time.Now().UnixNano())

	ret, err := CheckIndexExist(es, idx)
	require.Nil(t, err)
	require.False(t, ret)

	err = CreateIndex(es, idx)
	require.Nil(t, err)

	ret, err = CheckIndexExist(es, idx)
	require.Nil(t, err)
	require.True(t, ret)

	message := Article{
		ID:      987654321,
		Date:    123456789,
		Content: "看需求……\nns有主机和掌机模式\nlite是阉割轻量版，只有掌机模式",
	}
	err = StoreMessage(es, idx, &message)
	require.Nil(t, err)

	res, err := SearchMessage(es, idx, "主机", 0, 5)
	require.Nil(t, err)

	require.Equal(t, "看需求……\nns有<b>主机</b>和掌机模式", res.Hits.Hits[0].Highlight.Content[0])
}
