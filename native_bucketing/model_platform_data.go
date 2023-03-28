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

func (p *PlatformData) FromJSON(js []byte) (err error, rt PlatformData) {
	var clss PlatformData
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}
