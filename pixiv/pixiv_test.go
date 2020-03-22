package pixiv

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePixivURL(t *testing.T) {
	r := require.New(t)
	ret := ParseURL("♥ | NARU #pixiv https://www.pixiv.net/member_illust.php?illust_id=68698295&mode=medium")
	r.Equal(uint64(68698295), ret)

	ret = ParseURL("♥ | NARU #pixiv https://www.pixiv.net/member_illust.php?mode=medium&illust_id=68698295")
	r.Equal(uint64(68698295), ret)
}
