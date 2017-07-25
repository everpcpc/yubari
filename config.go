package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Config struct {
	File      string
	SelfQQ    string `json:"self_qq"`
	BotQQ     string `json:"bot_qq"`
	QQGroup   string `json:"qq_group"`
	BtdServer string `json:"btd_server"`
}

func ReadConfig(cfgfile *string) (cfg *Config) {
	cfg = &Config{File: *cfgfile}
	file, e := ioutil.ReadFile(*cfgfile)
	if e != nil {
		Logger.Fatalf("Configfile error (%v)\n", e)
		os.Exit(2)
	}
	json.Unmarshal(file, cfg)
	return cfg
}
