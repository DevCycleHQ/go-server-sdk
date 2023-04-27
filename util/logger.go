package util

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

var (
	globalLogger Logger = defaultLogger{}
	globalLock   sync.Mutex
)

func SetLogger(log Logger) {
	if log == nil {
		panic("Can't set the logger to nil")
	}

	globalLock.Lock()
	globalLogger = log
	globalLock.Unlock()
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

type Logger interface {
	// Printf - Straight print passthrough
	Printf(format string, a ...any)
	// Infof - Info level print
	Infof(format string, a ...any)
	// Debugf - Debug level print, mostly used for information/tracing
	Debugf(format string, a ...any)
	// Warnf - Warn level print, something that might be a problem
	Warnf(format string, a ...any)
	// Errorf - Error level print - returns an error
	Errorf(format string, a ...any) error
}

type defaultLogger struct{}

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

func (defaultLogger) Printf(format string, a ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	log.Printf(format, a...)
}

func (defaultLogger) Warnf(format string, a ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	log.Printf("WARN: "+format, a...)
}

func (defaultLogger) Errorf(format string, a ...any) error {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	log.Printf("ERROR: "+format, a...)
	return fmt.Errorf(format, a...)
}

type DiscardLogger struct{}

func (DiscardLogger) Printf(_ string, _ ...any) {

}

func (DiscardLogger) Infof(_ string, _ ...any) {

}

func (DiscardLogger) Debugf(_ string, _ ...any) {

}

func (DiscardLogger) Warnf(_ string, _ ...any) {

}

func (DiscardLogger) Errorf(_ string, _ ...any) error {
	return nil
}
