package main

import (
	"github.com/op/go-logging"
	"os"
)

var (
	Logger = GetLogger(true, false)
)

func GetLogger(std bool, sys bool) *logging.Logger {
	log := logging.MustGetLogger("yubari")
	if std {
		stdFormat := logging.MustStringFormatter(
			`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s}%{color:reset} %{message}`,
		)
		stdLogger := logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), stdFormat)
		logging.SetBackend(stdLogger)
	}
	if sys {
		sysFormat := logging.MustStringFormatter(
			`%{shortfunc} ▶ %{level:.4s} %{message}`,
		)
		_sysLogger, _ := logging.NewSyslogBackend("yubari")
		sysLogger := logging.NewBackendFormatter(_sysLogger, sysFormat)
		logging.SetBackend(sysLogger)
	}

	// logging.AddModuleLevel(stdBackend).SetLevel(logging.CRITICAL, "")
	// logging.SetBackend(stdLogger, sysLogger)

	return log
}
