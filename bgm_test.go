package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSubjectTitleFromURL(t *testing.T) {
	assert.Equal(t, "3月のライオン", getSubjectTitleFromURL("https://bgm.tv/ep/648826"))
	assert.Equal(t, "放課後のプレアデス", getSubjectTitleFromURL("https://bgm.tv/subject/86517"))
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
