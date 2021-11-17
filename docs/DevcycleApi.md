# {{classname}}

All URIs are relative to *https://bucketing-api.devcycle.com/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetFeatures**](DevcycleApi.md#GetFeatures) | **Post** /v1/features | Get all features by key for user data
[**GetVariableByKey**](DevcycleApi.md#GetVariableByKey) | **Post** /v1/variables/{key} | Get variable by key for user data
[**GetVariables**](DevcycleApi.md#GetVariables) | **Post** /v1/variables | Get all variables by key for user data
[**PostEvents**](DevcycleApi.md#PostEvents) | **Post** /v1/track | Post events to DevCycle for user

# **GetFeatures**
> map[string]Feature GetFeatures(ctx, body)
Get all features by key for user data

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**UserData**](UserData.md)|  | 

### Return type

[**map[string]Feature**](Feature.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetVariableByKey**
> Variable GetVariableByKey(ctx, body, key)
Get variable by key for user data

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**UserData**](UserData.md)|  | 
  **key** | **string**| Variable key | 

### Return type

[**Variable**](Variable.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetVariables**
> map[string]Variable GetVariables(ctx, body)
Get all variables by key for user data

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**UserData**](UserData.md)|  | 

### Return type

[**map[string]Variable**](Variable.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PostEvents**
> InlineResponse201 PostEvents(ctx, body)
Post events to DevCycle for user

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**UserDataAndEventsBody**](UserDataAndEventsBody.md)|  | 

### Return type

[**InlineResponse201**](inline_response_201.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

