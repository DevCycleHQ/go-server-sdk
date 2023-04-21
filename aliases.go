package devcycle

import (
	"github.com/devcyclehq/go-server-sdk/v2/api"
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

// Aliases to support customizing logging
type Logger = util.Logger
type DiscardLogger = util.DiscardLogger

func SetLogger(log Logger) { util.SetLogger(log) }

// Aliases for old DVC-prefixed types
type DVCOptions = Options
type DVCClient = Client
type DVCUser = api.User
type DVCEvent = api.Event

func NewDVCClient(sdkKey string, options *Options) (*DVCClient, error) {
	return NewClient(sdkKey, options)
}
