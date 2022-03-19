package rss

import (
	"testing"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDoubanUpdate(t *testing.T) {
	item := &gofeed.Item{
		Title:       "看过蜘蛛侠：英雄无归",
		Description: "\n\n    <table><tr>\n    <td width=\"80px\"><a href=\"https://movie.douban.com/subject/26933210/\" title=\"Spider-Man: No Way Home\">\n    <img src=\"https://img9.doubanio.com/view/photo/s_ratio_poster/public/p2730024046.jpg\" alt=\"Spider-Man: No Way Home\"></a></td>\n    <td>\n    </td></tr></table>\n",
		Content:     "",
		Link:        "http://movie.douban.com/subject/26933210/",
		Links:       []string{"http://movie.douban.com/subject/26933210/"},
		Published:   "Sun, 13 Mar 2022 16:02:55 GMT",
		GUID:        "https://www.douban.com/people/everpcpc/interests/3272542529",
	}
	output, err := getDoubanUpdate(item)
	require.Nil(t, err)
	assert.Equal(t,
		"看过蜘蛛侠：英雄无归「Spider-Man: No Way Home」\nhttps://movie.douban.com/subject/26933210/\n#Douban",
		output,
	)
}
