package devcycle

import (
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
)

type ErrorResponse = api.ErrorResponse
type BucketedUserConfig = api.BucketedUserConfig
type Environment = api.Environment
type BaseVariable = api.BaseVariable
type Variable = api.Variable
type ReadOnlyVariable = api.ReadOnlyVariable
type DVCUser = api.DVCUser
type DVCPopulatedUser = api.DVCPopulatedUser
type UserFeatureData = api.UserFeatureData
type UserDataAndEventsBody = api.UserDataAndEventsBody
type Project = api.Project
type ProjectSettings = api.ProjectSettings
type EdgeDBSettings = api.EdgeDBSettings
type OptInSettings = api.OptInSettings
type OptInColors = api.OptInColors
type PlatformData = api.PlatformData
type FeatureVariation = api.FeatureVariation
type DVCEvent = api.DVCEvent
type UserEventsBatchRecord = api.UserEventsBatchRecord
type FlushPayload = api.FlushPayload
type BatchEventsBody = api.BatchEventsBody
type Feature = api.Feature
type Logger = util.Logger
type DiscardLogger = util.DiscardLogger

func SetLogger(log Logger) { util.SetLogger(log) }
