package native_bucketing

import (
	"encoding/json"
	"gopkg.in/validator.v2"
)

type PlatformData struct {
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platformVersion"`
	SdkType         string `json:"sdkType"`
	SdkVersion      string `json:"sdkVersion"`
}

func (p *PlatformData) FromJSON(js string) (err error, rt PlatformData) {
	var clss PlatformData
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}
