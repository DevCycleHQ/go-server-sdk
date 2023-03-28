package native_bucketing

import (
	"encoding/json"
	"golang.org/x/exp/maps"
	"gopkg.in/validator.v2"
	"time"
)

type DVCUser struct {
	UserId            string                 `json:"user_id"`
	Email             string                 `json:"email"`
	Name              string                 `json:"name"`
	Language          string                 `json:"language"`
	Country           string                 `json:"country"`
	AppVersion        string                 `json:"appVersion"`
	AppBuild          float64                `json:"appBuild"`
	DeviceModel       string                 `json:"deviceModel"`
	CustomData        map[string]interface{} `json:"customData"`
	PrivateCustomData map[string]interface{} `json:"privateCustomData"`
}

func (u *DVCUser) FromJSON(js string) (err error, rt DVCUser) {
	var clss DVCUser
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type DVCPopulatedUser struct {
	UserId            string                 `json:"user_id"`
	Email             string                 `json:"email"`
	Name              string                 `json:"name"`
	Language          string                 `json:"language"`
	Country           string                 `json:"country"`
	AppVersion        string                 `json:"appVersion"`
	AppBuild          float64                `json:"appBuild"`
	DeviceModel       string                 `json:"deviceModel"`
	CustomData        map[string]interface{} `json:"customData"`
	PrivateCustomData map[string]interface{} `json:"privateCustomData"`
	CreatedDate       time.Time              `json:"createdDate"`
	LastSeenDate      time.Time              `json:"lastSeenDate"`
	Platform          string                 `json:"platform"`
	PlatformVersion   string                 `json:"platformVersion"`
	SdkType           string                 `json:"sdkType"`
	SdkVersion        string                 `json:"sdkVersion"`
}

func (u *DVCPopulatedUser) FromJSON(js string) (err error, rt DVCPopulatedUser) {
	var clss DVCPopulatedUser
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}
func (p *DVCPopulatedUser) CombinedCustomData() map[string]interface{} {
	var ret map[string]interface{}
	maps.Copy(ret, p.CustomData)
	maps.Copy(ret, p.PrivateCustomData)
	return ret
}
