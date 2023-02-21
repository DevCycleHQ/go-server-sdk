package devcycle

import (
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

func printf(format string, a ...any) {
	globalLogger.Printf(format, a...)
}

type Logger interface {
	Printf(format string, a ...any)
}

type defaultLogger struct{}

func (defaultLogger) Printf(format string, a ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	log.Printf(format, a...)
}

type DiscardLogger struct{}

func (DiscardLogger) Printf(_ string, _ ...any) {

}
