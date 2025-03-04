//go:build !devcycle_debug_logging

package util

func (defaultLogger) Debugf(format string, a ...any) {

}

func (defaultLogger) Infof(format string, a ...any) {

}
