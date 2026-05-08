package devcycle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheckDefaults_ZeroValues_AppliesDefaults(t *testing.T) {
	o := &Options{}
	o.CheckDefaults()

	assert.Equal(t, time.Second*30, o.EventFlushIntervalMS)
	assert.Equal(t, time.Second*10, o.ConfigPollingIntervalMS)
}

func TestCheckDefaults_OutOfRange_AppliesDefaults(t *testing.T) {
	o := &Options{
		EventFlushIntervalMS:    100 * time.Millisecond,
		ConfigPollingIntervalMS: 500 * time.Millisecond,
	}
	o.CheckDefaults()

	assert.Equal(t, time.Second*30, o.EventFlushIntervalMS)
	assert.Equal(t, time.Second*10, o.ConfigPollingIntervalMS)
}

func TestCheckDefaults_ValidValues_Unchanged(t *testing.T) {
	o := &Options{
		EventFlushIntervalMS:    time.Second * 10,
		ConfigPollingIntervalMS: time.Second * 5,
	}
	o.CheckDefaults()

	assert.Equal(t, time.Second*10, o.EventFlushIntervalMS)
	assert.Equal(t, time.Second*5, o.ConfigPollingIntervalMS)
}
