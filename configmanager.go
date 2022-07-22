package devcycle

import "time"

type EnvironmentConfigManager struct {
	EnvironmentKey  string
	PollingInterval time.Duration
	RequestTimeout  time.Duration
	configETag      string
	localBucketing  *DevCycleLocalBucketing
}

func (e *EnvironmentConfigManager) FetchConfig() {

}
