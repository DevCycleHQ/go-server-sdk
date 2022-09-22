package devcycle

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

func (pd *PlatformData) FromUser(user UserData) PlatformData {
	pd.Platform = user.Platform
	pd.SdkType = user.SdkType
	pd.SdkVersion = user.SdkVersion
	pd.PlatformVersion = user.PlatformVersion
	pd.DeviceModel = user.DeviceModel
	pd.Hostname, _ = os.Hostname()
	return *pd
}

func (pd *PlatformData) Default(isLocal bool) *PlatformData {
	pd.Platform = "golang"
	if isLocal {
		pd.SdkType = "local"
	} else {
		pd.SdkType = "cloud"
	}
	pd.SdkVersion = VERSION
	pd.PlatformVersion = runtime.Version()
	pd.Hostname, _ = os.Hostname()
	return pd
}
