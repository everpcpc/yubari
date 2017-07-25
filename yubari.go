package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
)

func main() {
	cfgfile := flag.String("c", "config.json", "Config file")
	flag.Parse()
	cfg := ReadConfig(cfgfile)
	fmt.Println(cfg)

	data := []byte(`{"/laugh": 12, "/cry": 2}`)
	var objmap map[string]*json.RawMessage
	err := json.Unmarshal(data, &objmap)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Logger.Critical("ttttt")

	faceId, err := strconv.Atoi(string(*objmap["/laugh"]))
	if err != nil {
		fmt.Println(err)
		return
	}
	face := QQface{faceId}
	fmt.Println(face.String())
	qqBot := &QQBot{Id: 0}
	err = qqBot.Connect("localhost:11300")
	if err != nil {
		fmt.Println(err)
		return
	}
	qqBot.SendSelfMsg("嗯？")
}
