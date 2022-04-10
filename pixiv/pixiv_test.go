package pixiv

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePixivURL(t *testing.T) {
	r := require.New(t)
	ret := ParseURL("♥ | NARU #pixiv https://www.pixiv.net/artworks/97336690")
	r.Equal(uint64(97336690), ret)

}
