package telegram

import (
	"fmt"
	"testing"
	"time"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/stretchr/testify/require"
)

func TestCreateIndex(t *testing.T) {
	es, err := elasticsearch7.NewDefaultClient()
	require.Nil(t, err)

	idx := fmt.Sprintf("test-%d", time.Now().UnixNano())

	ret, err := checkIndexExist(es, idx)
	require.Nil(t, err)
	require.False(t, ret)

	err = createIndex(es, idx)
	require.Nil(t, err)

	ret, err = checkIndexExist(es, idx)
	require.Nil(t, err)
	require.True(t, ret)
}
