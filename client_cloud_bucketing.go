package devcycle

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/matryer/try"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type cloudClient struct {
	cfg             *HTTPConfiguration
	DevCycleOptions *Options
	sdkKey          string
	platformData    *PlatformData
}

func newCloudClient(sdkKey string, options *Options, platformData *PlatformData) *cloudClient {
	cfg := NewConfiguration(options)
	c := &cloudClient{sdkKey: sdkKey}
	c.cfg = cfg
	c.DevCycleOptions = options
	c.platformData = platformData

	if c.DevCycleOptions.Logger != nil {
		util.SetLogger(c.DevCycleOptions.Logger)
	}

	return c
}

func (c *cloudClient) AllFeatures(user User) (map[string]Feature, error) {
	populatedUser := user.GetPopulatedUser(c.platformData)

	var (
		httpMethod          = strings.ToUpper("Post")
		postBody            interface{}
		localVarReturnValue map[string]Feature
	)

	// create path and map variables
	path := c.cfg.BasePath + "/v1/features"

	headers := make(map[string]string)
	queryParams := url.Values{}

	// body params
	postBody = &populatedUser

	r, rBody, err := c.performRequest(path, httpMethod, postBody, headers, queryParams)

	if err != nil {
		return nil, err
	}

	if r.StatusCode < 300 {
		// If we succeed, return the data, otherwise pass on to decode error.
		err = decode(&localVarReturnValue, rBody, r.Header.Get("Content-Type"))
		return localVarReturnValue, err
	}

	return nil, c.handleError(r, rBody)
}

func (c *cloudClient) VariableValue(userdata User, key string, defaultValue interface{}) (interface{}, error) {
	variable, err := c.Variable(userdata, key, defaultValue)
	return variable.Value, err
}

func (c *cloudClient) Variable(userdata User, key string, defaultValue interface{}) (result Variable, err error) {
	if key == "" {
		return Variable{}, errors.New("invalid key provided for call to Variable")
	}

	convertedDefaultValue := convertDefaultValueType(defaultValue)
	variableType, err := variableTypeFromValue(key, convertedDefaultValue, false)

	if err != nil {
		return Variable{}, err
	}

	baseVar := BaseVariable{Key: key, Value: convertedDefaultValue, Type_: variableType}
	variable := Variable{BaseVariable: baseVar, DefaultValue: convertedDefaultValue, IsDefaulted: true}

	defer func() {
		if r := recover(); r != nil {
			// Return a usable default value in a panic situation
			result = variable
			err = fmt.Errorf("recovered from panic in Variable eval: %v ", r)
			util.Errorf("%v", err)
		}
	}()

	populatedUser := userdata.GetPopulatedUser(c.platformData)

	var (
		httpMethod          = strings.ToUpper("Post")
		postBody            interface{}
		localVarReturnValue Variable
	)

	// create path and map variables
	path := c.cfg.BasePath + "/v1/variables/{key}"
	path = strings.Replace(path, "{"+"key"+"}", fmt.Sprintf("%v", key), -1)

	headers := make(map[string]string)
	queryParams := url.Values{}

	// userdata params
	postBody = &populatedUser

	r, body, err := c.performRequest(path, httpMethod, postBody, headers, queryParams)

	if err != nil {
		return variable, err
	}

	if r.StatusCode < 300 {
		// If we succeed, return the data, otherwise pass on to decode error.
		err = decode(&localVarReturnValue, body, r.Header.Get("Content-Type"))
		if err == nil && localVarReturnValue.Value != nil {
			if compareTypes(localVarReturnValue.Value, convertedDefaultValue) {
				variable.Value = localVarReturnValue.Value
				variable.IsDefaulted = false
			} else {
				util.Warnf("Type mismatch for variable %s. Expected type %s, got %s",
					key,
					reflect.TypeOf(defaultValue).String(),
					reflect.TypeOf(localVarReturnValue.Value).String(),
				)
			}

			return variable, err
		}
	}

	var v ErrorResponse
	err = decode(&v, body, r.Header.Get("Content-Type"))
	if err != nil {
		util.Warnf("Error decoding response body %s", err)
		return variable, nil
	}
	util.Warnf(v.Message)
	return variable, nil
}

func (c *cloudClient) AllVariables(user User) (map[string]ReadOnlyVariable, error) {
	var (
		httpMethod          = strings.ToUpper("Post")
		postBody            interface{}
		localVarReturnValue map[string]ReadOnlyVariable
	)

	populatedUser := user.GetPopulatedUser(c.platformData)

	// create path and map variables
	path := c.cfg.BasePath + "/v1/variables"

	headers := make(map[string]string)
	queryParams := url.Values{}

	// body params
	postBody = &populatedUser

	r, rBody, err := c.performRequest(path, httpMethod, postBody, headers, queryParams)
	if err != nil {
		return localVarReturnValue, err
	}

	if r.StatusCode < 300 {
		// If we succeed, return the data, otherwise pass on to decode error.
		err = decode(&localVarReturnValue, rBody, r.Header.Get("Content-Type"))
		return localVarReturnValue, err
	}

	return nil, c.handleError(r, rBody)
}

func (c *cloudClient) Track(user User, event Event) (bool, error) {
	if c.DevCycleOptions.DisableCustomEventLogging {
		return true, nil
	}
	if event.Type_ == "" {
		return false, errors.New("event type is required")
	}

	var (
		httpMethod = strings.ToUpper("Post")
		postBody   interface{}
	)

	populatedUser := user.GetPopulatedUser(c.platformData)

	events := []Event{event}
	body := UserDataAndEventsBody{User: &populatedUser, Events: events}
	// create path and map variables
	path := c.cfg.BasePath + "/v1/track"

	headers := make(map[string]string)
	queryParams := url.Values{}

	// body params
	postBody = &body

	r, rBody, err := c.performRequest(path, httpMethod, postBody, headers, queryParams)
	if err != nil {
		return false, err
	}

	if r.StatusCode < 300 {
		// If we succeed, return the data, otherwise pass on to decode error.
		err = decode(nil, rBody, r.Header.Get("Content-Type"))
		if err == nil {
			return false, err
		} else {
			return true, nil
		}
	}

	return false, c.handleError(r, rBody)
}

func (c *cloudClient) performRequest(
	path string, method string,
	postBody interface{},
	headerParams map[string]string,
	queryParams url.Values,
) (response *http.Response, body []byte, err error) {
	headerParams["Content-Type"] = "application/json"
	headerParams["Accept"] = "application/json"
	headerParams["Authorization"] = c.sdkKey

	var httpResponse *http.Response
	var responseBody []byte

	// This retrying lib works by retrying as long as the bool is true and err is not nil
	// the attempt param is auto-incremented
	err = try.Do(func(attempt int) (bool, error) {
		var err error
		r, err := c.prepareRequest(
			path,
			method,
			postBody,
			headerParams,
			queryParams,
		)

		// Don't retry if theres an error preparing the request
		if err != nil {
			return false, err
		}

		httpResponse, err = c.callAPI(r)
		if httpResponse == nil && err == nil {
			err = errors.New("Nil httpResponse")
		}
		if err != nil {
			time.Sleep(time.Duration(exponentialBackoff(attempt)) * time.Millisecond) // wait with exponential backoff
			return attempt <= 5, err
		}
		responseBody, err = io.ReadAll(httpResponse.Body)
		httpResponse.Body.Close()

		if err == nil && httpResponse.StatusCode >= 500 && attempt <= 5 {
			err = errors.New("5xx error on request")
		}

		if err != nil {
			time.Sleep(time.Duration(exponentialBackoff(attempt)) * time.Millisecond) // wait with exponential backoff
		}

		return attempt <= 5, err // try 5 times
	})

	if err != nil {
		return nil, nil, err
	}
	return httpResponse, responseBody, err

}

func (c *cloudClient) handleError(r *http.Response, body []byte) (err error) {
	newErr := GenericError{
		body:  body,
		error: r.Status,
	}

	var v ErrorResponse
	if len(body) > 0 {
		err = decode(&v, body, r.Header.Get("Content-Type"))
		if err != nil {
			newErr.error = err.Error()
			return newErr
		}
	}
	newErr.model = v

	if r.StatusCode >= 500 {
		util.Warnf("Server reported a 5xx error: ", newErr)
		return nil
	}
	return newErr
}

// callAPI do the request.
func (c *cloudClient) callAPI(request *http.Request) (*http.Response, error) {
	return c.cfg.HTTPClient.Do(request)
}

// Change base path to allow switching to mocks
func (c *cloudClient) ChangeBasePath(path string) {
	c.cfg.BasePath = path
}

func (c *cloudClient) SetOptions(dvcOptions Options) {
	c.DevCycleOptions = &dvcOptions
}

// prepareRequest build the request
func (c *cloudClient) prepareRequest(
	path string,
	method string,
	postBody interface{},
	headerParams map[string]string,
	queryParams url.Values,
) (localVarRequest *http.Request, err error) {

	var body *bytes.Buffer

	// Detect postBody type and post.
	if postBody != nil {
		contentType := headerParams["Content-Type"]
		if contentType == "" {
			contentType = detectContentType(postBody)
			headerParams["Content-Type"] = contentType
		}

		body, err = setBody(postBody, contentType)
		if err != nil {
			return nil, err
		}
	}

	// Setup path and query parameters
	builtURL, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	// Adding Query Param
	query := builtURL.Query()
	for k, v := range queryParams {
		for _, iv := range v {
			query.Add(k, iv)
		}
	}

	if c.DevCycleOptions.EnableEdgeDB {
		query.Add("enableEdgeDB", "true")
	}

	// Encode the parameters.
	builtURL.RawQuery = query.Encode()

	// Generate a new request
	if body != nil {
		localVarRequest, err = http.NewRequest(method, builtURL.String(), body)
	} else {
		localVarRequest, err = http.NewRequest(method, builtURL.String(), nil)
	}
	if err != nil {
		return nil, err
	}

	// add header parameters, if any
	if len(headerParams) > 0 {
		headers := http.Header{}
		for h, v := range headerParams {
			headers.Set(h, v)
		}
		localVarRequest.Header = headers
	}

	// Override request host, if applicable
	if c.cfg.Host != "" {
		localVarRequest.Host = c.cfg.Host
	}

	// Add the user agent to the request.
	localVarRequest.Header.Add("User-Agent", c.cfg.UserAgent)

	for header, value := range c.cfg.DefaultHeader {
		localVarRequest.Header.Add(header, value)
	}

	return localVarRequest, nil
}

func exponentialBackoff(attempt int) float64 {
	delay := math.Pow(2, float64(attempt)) * 100
	randomSum := delay * 0.2 * rand.Float64()
	return (delay + randomSum)
}
