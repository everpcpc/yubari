package bangumi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSubjectTitleFromURL(t *testing.T) {
	ret, err := getSubjectTitleFromURL("https://bgm.tv/ep/648826")
	assert.Nil(t, err)
	assert.Equal(t, "3月のライオン", ret)

	ret, err = getSubjectTitleFromURL("https://bgm.tv/subject/86517")
	assert.Nil(t, err)
	assert.Equal(t, "放課後のプレアデス", ret)
}

func TestGetBangumiUpdate(t *testing.T) {
	assert.Equal(t,
		"(❛ ◡ ❛)❤ 想看「Phantom in the Twilight」 https://bgm.tv/subject/241264 #Bangumi",
		getBangumiUpdate("想看 Phantom in the Twilight", "https://bgm.tv/subject/241264"),
	)
	assert.Equal(t,
		"♪(๑ᴖ◡ᴖ๑)♪ 读过「お兄ちゃんはおしまい！」第14话 https://bgm.tv/subject/243020 #Bangumi",
		getBangumiUpdate("读过 お兄ちゃんはおしまい！  第14话", "https://bgm.tv/subject/243020"),
	)
}
