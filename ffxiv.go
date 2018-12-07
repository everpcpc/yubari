package main

import (
	"fmt"
	"net/http"
)

type Quest struct {
	ID int
}
type Boss struct {
	ID    int
	Quest Quest
}
type Job struct {
	ID       int
	Name     string
	CNName   string
	NickName []string
}

func crawlDPS(boss Boss, job Job, day int) {
	fflogsURL := fmt.Sprintf("https://www.fflogs.com/zone/statistics/table/%d/dps/%d/100/8/1/100/1000/7/0/Global/%s/All/0/normalized/single/0/-1/", boss.Quest.ID, boss.ID, job.Name)
	http.Get(fflogsURL)
}

func onDPS(args string) (string, error) {
	return "", nil
}
