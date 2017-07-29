package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
)

func TestQQFace(t *testing.T) {
	data := []byte(`{"/laugh": 12, "/cry": 2}`)
	var objmap map[string]*json.RawMessage
	err := json.Unmarshal(data, &objmap)
	if err != nil {
		fmt.Println(err)
		return
	}
	faceID, err := strconv.Atoi(string(*objmap["/laugh"]))
	if err != nil {
		fmt.Println(err)
		return
	}
	face := QQFace(faceID)
	fmt.Println(face)
}

func TestQQImage(t *testing.T) {
	img := QQImage{"t.jpg"}
	fmt.Println(img)
}
