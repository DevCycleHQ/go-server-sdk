package native_bucketing

type PlatformData struct {
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platformVersion"`
	SdkType         string `json:"sdkType"`
	SdkVersion      string `json:"sdkVersion"`
}
