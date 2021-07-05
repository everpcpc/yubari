package main

import (
	"log/syslog"
	"os"

	"github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"
)

var (
	logger *logrus.Logger
)

func GetLogger(name, level string, logsys bool) *logrus.Logger {
	log := logrus.New()
	log.Out = os.Stdout
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	})

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		panic(err)
	}
	log.SetLevel(lvl)

	if logsys {
		hook, err := lSyslog.NewSyslogHook("", "", syslog.LOG_INFO, name)
		if err != nil {
			panic(err)
		}
		log.AddHook(hook)
	}

	return log
}
