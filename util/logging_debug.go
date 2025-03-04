//go:build devcycle_debug_logging

package util

import (
	"log"
	"strings"
)

func (defaultLogger) Debugf(format string, a ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	log.Printf("DEBUG: "+format, a...)
}

func (defaultLogger) Infof(format string, a ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	log.Printf("INFO: "+format, a...)
}
