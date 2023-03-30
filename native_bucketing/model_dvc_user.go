package native_bucketing

import (
	"golang.org/x/exp/maps"
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
	PlatformData
}

func (p *DVCPopulatedUser) CombinedCustomData() map[string]interface{} {
	var ret map[string]interface{}
	maps.Copy(ret, p.CustomData)
	maps.Copy(ret, p.PrivateCustomData)
	return ret
}
