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

// TODO: Set SDK version
func (pd *PlatformData) Default() *PlatformData {
	pd.Platform = "Go"
	pd.SdkType = "server"
	pd.PlatformVersion = runtime.Version()
	pd.Hostname, _ = os.Hostname()
	return pd
}