package devcycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"reflect"
)

var (
	_ context.Context
)

type DVCClientService service

func (a *DVCClientService) generateBucketedConfig(body DVCUser) (user BucketedUserConfig, err error) {
	userJSON, err := json.Marshal(body)
	if err != nil {
		return BucketedUserConfig{}, err
	}
	user, err = a.client.localBucketing.GenerateBucketedConfigForUser(string(userJSON))
	if err != nil {
		return BucketedUserConfig{}, err
	}
	user.user = &body
	return
}

func (a *DVCClientService) queueEvent(user DVCUser, event DVCEvent) (err error) {
	err = a.client.eventQueue.QueueEvent(user, event)
	return
}

func (a *DVCClientService) queueAggregateEvent(bucketed BucketedUserConfig, event DVCEvent) (err error) {
	err = a.client.eventQueue.QueueAggregateEvent(bucketed, event)
	return
}

/*
DVCClientService Get all features by key for user data
  - @param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
  - @param body

@return map[string]Feature
*/
func (a *DVCClientService) AllFeatures(ctx context.Context, body DVCUser) (map[string]Feature, error) {

	if !a.client.DevCycleOptions.EnableCloudBucketing {
		user, err := a.generateBucketedConfig(body)
		return user.Features, err
	}
	var (
		httpMethod          = strings.ToUpper("Post")
		postBody            interface{}
		localVarReturnValue map[string]Feature
	)

	// create path and map variables
	path := a.client.cfg.BasePath + "/v1/features"

	headers := make(map[string]string)
	queryParams := url.Values{}

	// body params
	postBody = &body
	if ctx != nil {
		// API Key Authentication
		if auth, ok := ctx.Value(ContextAPIKey).(APIKey); ok {
			var key string
			if auth.Prefix != "" {
				key = auth.Prefix + " " + auth.Key
			} else {
				key = auth.Key
			}
			headers["Authorization"] = key

		}
	}

	r, rBody, err := a.performRequest(ctx, path, httpMethod, postBody, headers, queryParams)

	if err != nil {
		return nil, err
	}

	if r.StatusCode < 300 {
		// If we succeed, return the data, otherwise pass on to decode error.
		err = a.client.decode(&localVarReturnValue, rBody, r.Header.Get("Content-Type"))
		return localVarReturnValue, err
	}

	return nil, a.handleError(r, rBody)
}

/*
DVCClientService Get variable by key for user data
  - @param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
  - @param body
  - @param key Variable key

@return Variable
*/
func (a *DVCClientService) Variable(ctx context.Context, userdata DVCUser, key string, defaultValue interface{}) (Variable, error) {
	readOnlyVariable := ReadOnlyVariable{Key: key, Value: convertDefaultValueType(defaultValue)}
	variable := Variable{ReadOnlyVariable: readOnlyVariable, DefaultValue: defaultValue, IsDefaulted: true}

	if !a.client.DevCycleOptions.EnableCloudBucketing {
		bucketed, err := a.generateBucketedConfig(userdata)

		variableEvaluationType := ""
		if bucketed.Variables[key].Value != nil && compareTypes(bucketed.Variables[key].Value, readOnlyVariable.Value) {
			variable.Value = bucketed.Variables[key].Value
			variable.IsDefaulted = false
			variableEvaluationType = EventType_AggVariableEvaluated
		} else {
			variableEvaluationType = EventType_AggVariableDefaulted
		}
		if !a.client.DevCycleOptions.DisableAutomaticEventLogging {
			err = a.queueAggregateEvent(bucketed, DVCEvent{
				Type_:  variableEvaluationType,
				Target: key,
			})
			if err != nil {
				log.Println("Error queuing aggregate event: ", err)
				err = nil
			}
		}
		return variable, err
	}

	var (
		httpMethod          = strings.ToUpper("Post")
		postBody            interface{}
		localVarReturnValue Variable
	)

	// create path and map variables
	path := a.client.cfg.BasePath + "/v1/variables/{key}"
	path = strings.Replace(path, "{"+"key"+"}", fmt.Sprintf("%v", key), -1)

	headers := make(map[string]string)
	queryParams := url.Values{}

	// userdata params
	postBody = &userdata

	r, body, err := a.performRequest(ctx, path, httpMethod, postBody, headers, queryParams)

	if err != nil {
		return localVarReturnValue, err
	}

	if r.StatusCode < 300 {
		// If we succeed, return the data, otherwise pass on to decode error.
		err = a.client.decode(&localVarReturnValue, body, r.Header.Get("Content-Type"))
		if err == nil {
			return localVarReturnValue, err
		}
	}

	var v ErrorResponse
	err = a.client.decode(&v, body, r.Header.Get("Content-Type"))
	if err != nil {
		log.Println(err.Error())
		return variable, nil
	}
	log.Println(v.Message)
	return variable, nil
}

func (a *DVCClientService) AllVariables(ctx context.Context, body DVCUser) (map[string]ReadOnlyVariable, error) {

	var (
		httpMethod          = strings.ToUpper("Post")
		postBody            interface{}
		localVarReturnValue map[string]ReadOnlyVariable
	)
	if !a.client.DevCycleOptions.EnableCloudBucketing {
		user, err := a.generateBucketedConfig(body)
		if err != nil {
			return localVarReturnValue, err
		}
		return user.Variables, err
	}

	// create path and map variables
	path := a.client.cfg.BasePath + "/v1/variables"

	headers := make(map[string]string)
	queryParams := url.Values{}

	// body params
	postBody = &body

	r, rBody, err := a.performRequest(ctx, path, httpMethod, postBody, headers, queryParams)
	if err != nil {
		return localVarReturnValue, err
	}

	if r.StatusCode < 300 {
		// If we succeed, return the data, otherwise pass on to decode error.
		err = a.client.decode(&localVarReturnValue, rBody, r.Header.Get("Content-Type"))
		return localVarReturnValue, err
	}

	return nil, a.handleError(r, rBody)
}

/*
DVCClientService Post events to DevCycle for user
  - @param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
  - @param body

@return InlineResponse201
*/

func (a *DVCClientService) Track(ctx context.Context, user DVCUser, event DVCEvent) (bool, error) {
	if a.client.DevCycleOptions.DisableCustomEventLogging {
		return true, nil
	}
	if event.Type_ == "" {
		return false, errors.New("event type is required")
	}

	if !a.client.DevCycleOptions.EnableCloudBucketing {
		err := a.client.eventQueue.QueueEvent(user, event)
		return err == nil, err
	}

	var (
		httpMethod = strings.ToUpper("Post")
		postBody   interface{}
	)

	events := []DVCEvent{event}
	body := UserDataAndEventsBody{User: &user, Events: events}
	// create path and map variables
	path := a.client.cfg.BasePath + "/v1/track"

	headers := make(map[string]string)
	queryParams := url.Values{}

	// body params
	postBody = &body

	r, rBody, err := a.performRequest(ctx, path, httpMethod, postBody, headers, queryParams)
	if err != nil {
		return false, err
	}

	if r.StatusCode < 300 {
		// If we succeed, return the data, otherwise pass on to decode error.
		err = a.client.decode(nil, rBody, r.Header.Get("Content-Type"))
		if err == nil {
			return false, err
		} else {
			return true, nil
		}
	}

	return false, a.handleError(r, rBody)
}

func (a *DVCClientService) FlushEvents() error {

	if a.client.DevCycleOptions.EnableCloudBucketing {
		return nil
	}

	if a.client.DevCycleOptions.DisableCustomEventLogging && a.client.DevCycleOptions.DisableAutomaticEventLogging {
		return nil
	}

	err := a.client.eventQueue.FlushEvents()
	return err
}

/*
Close the client and flush any pending events. Stop any ongoing tickers
*/
func (a *DVCClientService) Close() (err error) {
	if a.client.DevCycleOptions.EnableCloudBucketing {
		return
	}

	err = a.client.eventQueue.Close()
	a.client.configManager.Close()
	return err
}

func (a *DVCClientService) performRequest(
	ctx context.Context,
	path string, method string,
	postBody interface{},
	headerParams map[string]string,
	queryParams url.Values,
) (response *http.Response, body []byte, err error) {
	headerParams["Content-Type"] = "application/json"
	headerParams["Accept"] = "application/json"

	if ctx != nil {
		// API Key Authentication
		if auth, ok := ctx.Value(ContextAPIKey).(APIKey); ok {
			var key string
			if auth.Prefix != "" {
				key = auth.Prefix + " " + auth.Key
			} else {
				key = auth.Key
			}
			headerParams["Authorization"] = key

		}
	}

	r, err := a.client.prepareRequest(
		ctx,
		path,
		method,
		postBody,
		headerParams,
		queryParams,
	)

	if err != nil {
		return nil, nil, err
	}

	httpResponse, err := a.client.callAPI(r)
	if err != nil || httpResponse == nil {
		return nil, nil, err
	}

	responseBody, err := ioutil.ReadAll(httpResponse.Body)
	httpResponse.Body.Close()

	if err != nil {
		return nil, nil, err
	}

	return httpResponse, responseBody, err
}

func (a *DVCClientService) handleError(r *http.Response, body []byte) (err error) {
	newErr := GenericSwaggerError{
		body:  body,
		error: r.Status,
	}

	var v ErrorResponse
	err = a.client.decode(&v, body, r.Header.Get("Content-Type"))
	if err != nil {
		newErr.error = err.Error()
		return newErr
	}
	newErr.model = v
	return newErr
}

func compareTypes(value1 interface{}, value2 interface{}) bool {
	return reflect.TypeOf(value1) == reflect.TypeOf(value2)
}

func convertDefaultValueType(value interface{}) interface{} {
	switch value.(type) {
	case int:
		return float64(value.(int))
	case int8:
		return float64(value.(int8))
	case int16:
		return float64(value.(int16))
	case int32:
		return float64(value.(int32))
	case int64:
		return float64(value.(int64))
	case uint:
		return float64(value.(uint))
	case uint8:
		return float64(value.(uint8))
	case uint16:
		return float64(value.(uint16))
	case uint32:
		return float64(value.(uint32))
	case uint64:
		return float64(value.(uint64))
	case float32:
		return float64(value.(float32))
	default:
		return value
	}
}
