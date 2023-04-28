package api

import (
	"os"
	"runtime"
)

type PlatformData struct {
	SdkType         string `json:"sdkType"`
	SdkVersion      string `json:"sdkVersion"`
	PlatformVersion string `json:"platformVersion"`
	DeviceModel     string `json:"deviceModel"`
	Platform        string `json:"platform"`
	Hostname        string `json:"hostname"`
}

func (p PlatformData) Default() *PlatformData {
	var err error
	p.SdkType = "server"
	p.PlatformVersion = runtime.Version()
	p.Platform = "Go"
	p.Hostname, err = os.Hostname()
	if err != nil {
		p.Hostname = "aggregate"
	}

	return &p
}
