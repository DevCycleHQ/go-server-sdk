//go:build devcycle_debug_logging

package util

import (
	"log"
)

func init() {
	log.Printf("DevCycle debug logging enabled")
}

func Printf(format string, a ...any) {
	globalLogger.Printf(format, a...)
}

func Infof(format string, a ...any) {
	globalLogger.Infof(format, a...)
}

func Debugf(format string, a ...any) {
	globalLogger.Debugf(format, a...)
}

func Warnf(format string, a ...any) {
	globalLogger.Warnf(format, a...)
}

func Errorf(format string, a ...any) error {
	return globalLogger.Errorf(format, a...)
}
