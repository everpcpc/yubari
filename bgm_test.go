package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSubjectFromURL(t *testing.T) {
	assert.Equal(t, "3月のライオン", getSubjectFromEP("https://bgm.tv/ep/648826"))
}

func TestGetBangumiUpdate(t *testing.T) {
	assert.Equal(t,
		"(❛ ◡ ❛)❤ 想看「Phantom in the Twilight」https://bgm.tv/subject/241264 #Bangumi",
		getBangumiUpdate("想看 Phantom in the Twilight", "https://bgm.tv/subject/241264"),
	)
}
