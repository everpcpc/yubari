package yubari

import (
	"github.com/op/go-logging"
	"os"
)

var (
	Logger = GetLogger()
)

func GetLogger() *logging.Logger {
	log := logging.MustGetLogger("yubari")

	stdFormat := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{color:reset} %{message}`,
	)
	sysFormat := logging.MustStringFormatter(
		`%{shortfunc} ▶ %{level:.4s} %{message}`,
	)

	_sysLogger, _ := logging.NewSyslogBackend("yubari")
	sysLogger := logging.NewBackendFormatter(_sysLogger, sysFormat)
	stdLogger := logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), stdFormat)

	// logging.AddModuleLevel(stdBackend).SetLevel(logging.CRITICAL, "")
	logging.SetBackend(sysLogger, stdLogger)

	return log
}
