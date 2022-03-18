package rss

import (
	"testing"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSubjectTitleFromURL(t *testing.T) {
	ret, err := getBangumiSubjectTitleFromURL("https://bgm.tv/ep/648826")
	require.Nil(t, err)
	assert.Equal(t, "3月のライオン", ret)

	ret, err = getBangumiSubjectTitleFromURL("https://bgm.tv/subject/86517")
	require.Nil(t, err)
	assert.Equal(t, "放課後のプレアデス", ret)
}

func TestGetBangumiUpdate(t *testing.T) {
	item := &gofeed.Item{
		Title:       "读过 先生は恋を教えられない  第45话",
		Description: "\n读过 <a href=\"https://bgm.tv/subject/250377\" class=\"l\">先生は恋を教えられない</a>  第45话",
		Content:     "",
		Link:        "http://bgm.tv/user/everpcpc/timeline",
		Links:       []string{"http://bgm.tv/user/everpcpc/timeline"},
		Published:   "Fri, 18 Mar 2022 02:49:37 +0000",
		GUID:        "http://bgm.tv/user/everpcpc/timeline/27582014",
	}
	output, err := getBangumiUpdate(item)
	require.Nil(t, err)
	assert.Equal(t,
		"♪(๑ᴖ◡ᴖ๑)♪ 读过「先生は恋を教えられない」第45话 #Bangumi https://bgm.tv/subject/250377",
		output,
	)
}
