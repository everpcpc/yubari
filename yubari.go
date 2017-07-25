package main

import (
	"flag"
	"fmt"
)

func main() {
	cfgfile := flag.String("c", "config.json", "Config file")
	flag.Parse()
	cfg := ReadConfig(cfgfile)
	Logger.Infof("Starting with config: %+v", cfg)

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
	var err error
	qqBot, err = NewQQBot(cfg)
	if err != nil {
		fmt.Println(err)
		return
	}
	Logger.Infof("Starting qqbot: %+v", qqBot)
	qqWatch()
}

func qqWatch() {
	messages := make(chan map[string]string)
	go qqBot.Poll(messages)
	for msg := range messages {
		switch msg["event"] {
		case "PrivateMsg":
			Logger.Infof("[%s]:{%s}", msg["qq"], msg["msg"])
		case "GroupMsg":
			Logger.Infof("(%s)[%s]:{%s}", msg["group"], msg["qq"], msg["msg"])
		default:
			Logger.Info(msg)
		}
	}
}
