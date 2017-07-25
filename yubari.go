package main

import (
	"flag"
	"fmt"
	"time"
)

func main() {
	cfgfile := flag.String("c", "config.json", "Config file")
	flag.Parse()
	cfg := ReadConfig(cfgfile)
	fmt.Println(cfg)

	/*
		data := []byte(`{"/laugh": 12, "/cry": 2}`)
		var objmap map[string]*json.RawMessage
		err := json.Unmarshal(data, &objmap)
		if err != nil {
			fmt.Println(err)
			return
		}

		faceId, err := strconv.Atoi(string(*objmap["/laugh"]))
		if err != nil {
			fmt.Println(err)
			return
		}
		face := QQface{faceId}
		fmt.Println(face.String())
	*/

	qqBot := &QQBot{Id: cfg.BotQQ, Cfg: cfg}
	err := qqBot.Connect(cfg.BtdServer)
	if err != nil {
		fmt.Println(err)
		return
	}
	go qqBot.SendSelfMsg("嗯？")
	time.Sleep(1 * time.Second)
}
