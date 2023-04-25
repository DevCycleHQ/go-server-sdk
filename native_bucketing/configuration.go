package native_bucketing

import (
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"time"
)

type NativeBucketingConfiguration struct {
	FlushEventsInterval          time.Duration `json:"flushEventsMS"`
	DisableAutomaticEventLogging bool          `json:"disableAutomaticEventLogging"`
	DisableCustomEventLogging    bool          `json:"disableCustomEventLogging"`
}

var configuration = NativeBucketingConfiguration{}

var platformData = (api.PlatformData{}).Default()
