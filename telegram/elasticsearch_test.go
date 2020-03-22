package telegram

import (
	"testing"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/stretchr/testify/require"
)

func TestCreateIndex(t *testing.T) {
	es, err := elasticsearch7.NewDefaultClient()
	require.Nil(t, err)

	ret, err := checkIndexExist(es, "ttt")
	require.Nil(t, err)
	require.False(t, ret)

	err = createIndex(es, "ttt")
	require.Nil(t, err)

	ret, err = checkIndexExist(es, "ttt")
	require.Nil(t, err)
	require.True(t, ret)
}
