package devcycle

type PlatformData struct {
	SdkType         string `json:"sdkType"`
	SdkVersion      string `json:"sdkVersion"`
	PlatformVersion string `json:"platformVersion"`
	DeviceModel     string `json:"deviceModel"`
	Platform        string `json:"platform"`
}

func (pd *PlatformData) FromUser(user UserData) PlatformData {
	pd.Platform = user.Platform
	pd.SdkType = user.SdkType
	pd.SdkVersion = user.SdkVersion
	pd.PlatformVersion = user.PlatformVersion
	pd.DeviceModel = user.DeviceModel
	return *pd
}
