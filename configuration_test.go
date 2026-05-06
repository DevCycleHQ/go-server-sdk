package devcycle

import (
	"testing"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/stretchr/testify/assert"
)

type recordingLogger struct {
	warns []string
}

func (r *recordingLogger) Printf(_ string, _ ...any)         {}
func (r *recordingLogger) Infof(_ string, _ ...any)          {}
func (r *recordingLogger) Debugf(_ string, _ ...any)         {}
func (r *recordingLogger) Warnf(format string, a ...any)     { r.warns = append(r.warns, format) }
func (r *recordingLogger) Errorf(_ string, _ ...any) error   { return nil }

func TestCheckDefaults_ZeroValues_NoWarnings(t *testing.T) {
	logger := &recordingLogger{}
	util.SetLogger(logger)
	t.Cleanup(func() { util.SetLogger(util.DiscardLogger{}) })

	o := &Options{}
	o.CheckDefaults()

	assert.Empty(t, logger.warns, "expected no warnings for zero-valued options")
	assert.Equal(t, time.Second*30, o.EventFlushIntervalMS)
	assert.Equal(t, time.Second*10, o.ConfigPollingIntervalMS)
}

func TestCheckDefaults_OutOfRange_Warns(t *testing.T) {
	logger := &recordingLogger{}
	util.SetLogger(logger)
	t.Cleanup(func() { util.SetLogger(util.DiscardLogger{}) })

	o := &Options{
		EventFlushIntervalMS:    100 * time.Millisecond,
		ConfigPollingIntervalMS: 500 * time.Millisecond,
	}
	o.CheckDefaults()

	assert.Len(t, logger.warns, 2)
	assert.Equal(t, time.Second*30, o.EventFlushIntervalMS)
	assert.Equal(t, time.Second*10, o.ConfigPollingIntervalMS)
}

func TestCheckDefaults_ValidValues_NoWarnings(t *testing.T) {
	logger := &recordingLogger{}
	util.SetLogger(logger)
	t.Cleanup(func() { util.SetLogger(util.DiscardLogger{}) })

	o := &Options{
		EventFlushIntervalMS:    time.Second * 10,
		ConfigPollingIntervalMS: time.Second * 5,
	}
	o.CheckDefaults()

	assert.Empty(t, logger.warns)
	assert.Equal(t, time.Second*10, o.EventFlushIntervalMS)
	assert.Equal(t, time.Second*5, o.ConfigPollingIntervalMS)
}
