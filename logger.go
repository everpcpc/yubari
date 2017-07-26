package main

import (
	"github.com/op/go-logging"
	"os"
)

const (
	LOGALL = 0
	LOGSTD = 1
	LOGSYS = 2
)

var (
	logger *logging.Logger
)

func GetLogger(pos int) *logging.Logger {
	log := logging.MustGetLogger("yubari")
	stdFormat := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s}%{color:reset} %{message}`,
	)
	stdLogger := logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), stdFormat)
	sysFormat := logging.MustStringFormatter(
		`%{shortfunc} ▶ %{level:.4s} %{message}`,
	)
	_sysLogger, _ := logging.NewSyslogBackend("yubari")
	sysLogger := logging.NewBackendFormatter(_sysLogger, sysFormat)

	switch pos {
	case LOGALL:
		logging.SetBackend(stdLogger, sysLogger)
	case LOGSTD:
		logging.SetBackend(stdLogger)
	case LOGSYS:
		logging.SetBackend(sysLogger)
	default:
	}

	// logging.AddModuleLevel(stdBackend).SetLevel(logging.CRITICAL, "")

	return log
}
