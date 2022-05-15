package pixiv

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePixivURL(t *testing.T) {
	ret := ParseURL("♥ | NARU #pixiv https://www.pixiv.net/artworks/97336690")
	require.Equal(t, uint64(97336690), ret)
}
