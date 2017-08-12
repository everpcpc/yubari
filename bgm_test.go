package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSubjectFromEP(t *testing.T) {
	assert.Equal(t, "3月のライオン", getSubjectFromEP("https://bgm.tv/ep/648826"))
}
