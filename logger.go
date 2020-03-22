package main

import (
	"os"

	logging "github.com/op/go-logging"
)

const (
	LOGSYS   = 1 << 0
	LOGCOLOR = 1 << 1
	LOGTIME  = 1 << 2
)

var (
	logger *logging.Logger
)

func GetLogger(name, level string, flags byte) *logging.Logger {
	log := logging.MustGetLogger(name)
	formatString := `%{level:.4s} ▶ %{shortfunc} %{message}`
	if (flags & LOGTIME) != 0 {
		formatString = `%{time:15:04:05.000} ` + formatString
	}
	if (flags & LOGCOLOR) != 0 {
		formatString = `%{color}` + formatString + `%{color:reset}`
	}
	format := logging.MustStringFormatter(formatString)

	logger := logging.AddModuleLevel(
		logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "("+name+")", 0), format),
	)
	switch level {
	case "debug":
		logger.SetLevel(logging.DEBUG, "")
	case "info":
		logger.SetLevel(logging.INFO, "")
	case "notice":
		logger.SetLevel(logging.NOTICE, "")
	case "warning":
		logger.SetLevel(logging.WARNING, "")
	case "error":
		logger.SetLevel(logging.ERROR, "")
	default:
		logger.SetLevel(logging.INFO, "")
	}

	if (flags & LOGSYS) != 0 {
		_sysLogger, _ := logging.NewSyslogBackend(name)
		sysFormat := logging.MustStringFormatter(`%{level:.4s} %{shortfunc} ▶ %{message}`)
		sysLogger := logging.AddModuleLevel(logging.NewBackendFormatter(_sysLogger, sysFormat))
		sysLogger.SetLevel(logging.INFO, "")
		logging.SetBackend(logger, sysLogger)
	} else {
		logging.SetBackend(logger)
	}

	return log
}
