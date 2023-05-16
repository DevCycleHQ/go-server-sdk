//go:build !devcycle_debug_logging

package util

func Printf(format string, a ...any) {}

func Infof(format string, a ...any) {}

func Debugf(format string, a ...any) {}

func Warnf(format string, a ...any) {}

func Errorf(format string, a ...any) {}
