package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Config struct {
	File          string
	QQSelf        string   `json:"qq_self"`
	QQBot         string   `json:"qq_bot"`
	QQGroup       string   `json:"qq_group"`
	QQIgnore      []string `json:"qq_ignore"`
	BeanstalkAddr string   `json:"beanstalk_addr"`
}

func ReadConfig(cfgfile *string) (cfg *Config) {
	cfg = &Config{File: *cfgfile}
	file, e := ioutil.ReadFile(*cfgfile)
	if e != nil {
		logger.Fatalf("Configfile error (%v)\n", e)
		os.Exit(2)
	}
	json.Unmarshal(file, cfg)
	return cfg
}
