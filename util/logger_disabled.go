//go:build !devcycle_debug_logging

package util

func Printf(format string, a ...any) {}

func Infof(format string, a ...any) {}

func Debugf(format string, a ...any) {}

func Warnf(format string, a ...any) {}

// TODO: Remove this placeholder error once all calls to Errorf are removed from the hot paths
func Errorf(format string, a ...any) error { return placeholderErrorInstance }

type placeholderError struct{}

func (placeholderError) Error() string {
	return "Error in DevCycle SDK"
}

var placeholderErrorInstance = placeholderError{}
