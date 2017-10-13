package main

import (
	"github.com/op/go-logging"
	"os"
)

const (
	// LOGALL log to std and syslog
	LOGALL = 0
	// LOGSTD log only to std
	LOGSTD = 1
	// LOGSYS only log to syslog
	LOGSYS = 2
)

var (
	logger *logging.Logger
)

// GetLogger ...
func GetLogger(pos int) *logging.Logger {
	log := logging.MustGetLogger("yubari")
	stdFormat := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{level:.4s} ▶ %{shortfunc} %{message}%{color:reset}`,
	)
	stdLogger := logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), stdFormat)
	sysFormat := logging.MustStringFormatter(
		`%{level:.4s} ▶ %{shortfunc} %{message}`,
	)
	_sysLogger, _ := logging.NewSyslogBackend("yubari")
	sysLogger := logging.AddModuleLevel(logging.NewBackendFormatter(_sysLogger, sysFormat))
	sysLogger.SetLevel(logging.INFO, "")

	switch pos {
	case LOGALL:
		logging.SetBackend(stdLogger, sysLogger)
	case LOGSTD:
		logging.SetBackend(stdLogger)
	case LOGSYS:
		logging.SetBackend(sysLogger)
	default:
	}

	return log
}
