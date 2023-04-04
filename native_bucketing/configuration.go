package native_bucketing

import "time"

type NativeBucketingConfiguration struct {
	FlushEventsInterval          time.Duration `json:"flushEventsMS"`
	DisableAutomaticEventLogging bool          `json:"disableAutomaticEventLogging"`
	DisableCustomEventLogging    bool          `json:"disableCustomEventLogging"`
}

var configuration = NativeBucketingConfiguration{}
