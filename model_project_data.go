package devcycle

type PlatformData struct {
	SdkType         string `json:"sdkType"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platformVersion"`
	DeviceModel     string `json:"deviceModel"`
	SdkVersion      string `json:"sdkVersion"`
}
