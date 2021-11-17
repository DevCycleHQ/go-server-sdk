# UserData

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**UserId** | **string** | Unique id to identify the user | [default to null]
**Email** | **string** | User&#x27;s email used to identify the user on the dashboard / target audiences | [optional] [default to null]
**Name** | **string** | User&#x27;s name used to identify the user on the dashboard / target audiences | [optional] [default to null]
**Language** | **string** | User&#x27;s language in ISO 639-1 format | [optional] [default to null]
**Country** | **string** | User&#x27;s country in ISO 3166 alpha-2 format | [optional] [default to null]
**AppVersion** | **string** | App Version of the running application | [optional] [default to null]
**AppBuild** | **string** | App Build number of the running application | [optional] [default to null]
**CustomData** | [***interface{}**](interface{}.md) | User&#x27;s custom data to target the user with, data will be logged to DevCycle for use in dashboard. | [optional] [default to null]
**PrivateCustomData** | [***interface{}**](interface{}.md) | User&#x27;s custom data to target the user with, data will not be logged to DevCycle only used for feature bucketing. | [optional] [default to null]
**CreatedDate** | **float64** | Date the user was created, Unix epoch timestamp format | [optional] [default to null]
**LastSeenDate** | **float64** | Date the user was created, Unix epoch timestamp format | [optional] [default to null]
**Platform** | **string** | Platform the Client SDK is running on | [optional] [default to null]
**PlatformVersion** | **string** | Version of the platform the Client SDK is running on | [optional] [default to null]
**DeviceModel** | **string** | User&#x27;s device model | [optional] [default to null]
**SdkType** | **string** | DevCycle SDK type | [optional] [default to null]
**SdkVersion** | **string** | DevCycle SDK Version | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

