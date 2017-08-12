package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestQQFace(t *testing.T) {
	data := []byte(`{"/laugh": 12, "/cry": 2}`)
	var objmap map[string]*json.RawMessage

	err := json.Unmarshal(data, &objmap)
	assert.Nil(t, err)

	faceID, err := strconv.Atoi(string(*objmap["/laugh"]))
	assert.Nil(t, err)
	face := QQFace(faceID)
	assert.Equal(t, "[CQ:face,id=12]", face.String())
}

func TestQQImage(t *testing.T) {
	img := QQImage{"t.jpg"}
	assert.Equal(t, "[CQ:image,file=t.jpg]", img.String())
}
