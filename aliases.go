package devcycle

import (
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/bucketing"
	"github.com/devcyclehq/go-server-sdk/v2/util"
)

// Aliases for the types in the api package
type ErrorResponse = api.ErrorResponse
type BucketedUserConfig = api.BucketedUserConfig
type BaseVariable = api.BaseVariable
type Variable = api.Variable
type ReadOnlyVariable = api.ReadOnlyVariable
type User = api.User
type UserDataAndEventsBody = api.UserDataAndEventsBody
type PlatformData = api.PlatformData
type FeatureVariation = api.FeatureVariation
type Event = api.Event
type FlushPayload = api.FlushPayload
type BatchEventsBody = api.BatchEventsBody
type Feature = api.Feature

var ErrQueueFull = bucketing.ErrQueueFull

// Aliases to support customizing logging
type Logger = util.Logger
type DiscardLogger = util.DiscardLogger

func SetLogger(log Logger) { util.SetLogger(log) }

// Deprecated: Use devcycle.Options instead
type DVCOptions = Options

// Deprecated: Use devcycle.Client instead
type DVCClient = Client

// Deprecated: Use devcycle.User instead
type DVCUser = api.User

// Deprecated: Use devcycle.Event instead
type DVCEvent = api.Event

// Deprecated: Use devcycle.NewClient instead
func NewDVCClient(sdkKey string, options *Options) (*Client, error) {
	return NewClient(sdkKey, options)
}
