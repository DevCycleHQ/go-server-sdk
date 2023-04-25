package devcycle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/proto"
	"github.com/matryer/try"
)

var (
	jsonCheck = regexp.MustCompile("(?i:[application|text]/json)")
	xmlCheck  = regexp.MustCompile("(?i:[application|text]/xml)")
)

func GeneratePlatformData() *api.PlatformData {
	hostname, _ := os.Hostname()
	return &PlatformData{
		Platform:        "Go",
		SdkType:         "server",
		PlatformVersion: runtime.Version(),
		Hostname:        hostname,
		SdkVersion:      VERSION,
	}
}

// DVCClient
// In most cases there should be only one, shared, DVCClient.
type DVCClient struct {
	cfg             *HTTPConfiguration
	ctx             context.Context
	common          service // Reuse a single struct instead of allocating one for each service on the heap.
	DevCycleOptions *DVCOptions
	sdkKey          string
	configManager   *EnvironmentConfigManager
	eventQueue      EventQueuer
	localBucketing  LocalBucketing
	platformData    *PlatformData

	// Set to true when the client has been initialized, regardless of whether the config has loaded successfully.
	isInitialized                bool
	internalOnInitializedChannel chan bool
}

type LocalBucketing interface {
	ConfigReceiver
	GenerateBucketedConfigForUser(user DVCUser) (ret *BucketedUserConfig, err error)
	SetClientCustomData(map[string]interface{}) error
	Variable(user DVCUser, key string, variableType string) (variable Variable, err error)
	Close()
}

type EventQueuer interface {
	QueueEvent(user DVCUser, event DVCEvent) error
	QueueAggregateEvent(config BucketedUserConfig, event DVCEvent) error
	FlushEvents() (err error)
	Metrics() (int32, int32)
	Close() (err error)
}

type SDKEvent struct {
	Success             bool   `json:"success"`
	Message             string `json:"message"`
	Error               error  `json:"error"`
	FirstInitialization bool   `json:"firstInitialization"`
}

type service struct {
	client *DVCClient
}

// NewDVCClient creates a new API client.
// optionally pass a custom http.Client to allow for advanced features such as caching.
func NewDVCClient(sdkKey string, options *DVCOptions) (*DVCClient, error) {
	if sdkKey == "" {
		return nil, errorf("missing sdk key! Call NewDVCClient with a valid sdk key")
	}
	if !sdkKeyIsValid(sdkKey) {
		return nil, fmt.Errorf("Invalid sdk key. Call NewDVCClient with a valid sdk key.")
	}
	cfg := NewConfiguration(options)

	options.CheckDefaults()

	c := &DVCClient{sdkKey: sdkKey}
	c.cfg = cfg
	c.ctx = context.Background()
	c.common.client = c
	c.DevCycleOptions = options
	c.platformData = GeneratePlatformData()

	if c.DevCycleOptions.Logger != nil {
		SetLogger(c.DevCycleOptions.Logger)
	}

	if !c.DevCycleOptions.EnableCloudBucketing {
		c.internalOnInitializedChannel = make(chan bool, 1)

		err := c.setLBClient(sdkKey, options)
		if err != nil {
			return c, err
		}

		c.configManager = NewEnvironmentConfigManager(sdkKey, c.localBucketing, options, c.cfg)
		c.configManager.StartPolling(options.ConfigPollingIntervalMS)

		if c.DevCycleOptions.OnInitializedChannel != nil {
			// TODO: Pass this error back via a channel internally
			go func() {
				_ = c.configManager.initialFetch()
				c.handleInitialization()
			}()
		} else {
			err := c.configManager.initialFetch()
			c.handleInitialization()
			return c, err
		}
	} else if c.DevCycleOptions.OnInitializedChannel != nil {
		go func() {
			c.DevCycleOptions.OnInitializedChannel <- true
		}()
	}
	return c, nil
}

func (c *DVCClient) handleInitialization() {
	c.isInitialized = true
	if c.DevCycleOptions.OnInitializedChannel != nil {
		go func() {
			c.DevCycleOptions.OnInitializedChannel <- true
		}()

	}
	c.internalOnInitializedChannel <- true
}

func (c *DVCClient) generateBucketedConfig(user DVCUser) (config *BucketedUserConfig, err error) {
	config, err = c.localBucketing.GenerateBucketedConfigForUser(user)
	if err != nil {
		return nil, err
	}
	config.User = &user
	return
}

func createNullableString(val string) *proto.NullableString {
	if val == "" {
		return &proto.NullableString{Value: "", IsNull: true}
	} else {
		return &proto.NullableString{Value: val, IsNull: false}
	}
}

func createNullableDouble(val float64) *proto.NullableDouble {
	if math.IsNaN(val) {
		return &proto.NullableDouble{Value: 0, IsNull: true}
	} else {
		return &proto.NullableDouble{Value: val, IsNull: false}
	}
}

func createNullableCustomData(data map[string]interface{}) *proto.NullableCustomData {
	dataMap := map[string]*proto.CustomDataValue{}

	if len(data) == 0 {
		return &proto.NullableCustomData{
			Value:  dataMap,
			IsNull: true,
		}
	}
	// pull the values from the map and convert to the nullable data objects for protobuf
	for key, val := range data {
		if val == nil {
			dataMap[key] = &proto.CustomDataValue{Type: proto.CustomDataType_Null}
			continue
		}

		switch val := val.(type) {
		case string:
			dataMap[key] = &proto.CustomDataValue{Type: proto.CustomDataType_Str, StringValue: val}
		case float64:
			dataMap[key] = &proto.CustomDataValue{Type: proto.CustomDataType_Num, DoubleValue: val}
		case bool:
			dataMap[key] = &proto.CustomDataValue{Type: proto.CustomDataType_Bool, BoolValue: val}
		default:
			// if we don't know what it is, just set it to null
			dataMap[key] = &proto.CustomDataValue{Type: proto.CustomDataType_Null}
		}
	}

	return &proto.NullableCustomData{
		Value:  dataMap,
		IsNull: false,
	}
}

func (c *DVCClient) queueAggregateEvent(bucketed BucketedUserConfig, event DVCEvent) (err error) {
	err = c.eventQueue.QueueAggregateEvent(bucketed, event)
	return
}

/*
DVCClientService Get all features by key for user data
  - @param body

@return map[string]Feature
*/
func (c *DVCClient) AllFeatures(user DVCUser) (map[string]Feature, error) {
	if !c.DevCycleOptions.EnableCloudBucketing {
		if c.hasConfig() {
			user, err := c.generateBucketedConfig(user)
			if err != nil {
				return nil, fmt.Errorf("error generating bucketed config: %w", err)
			}
			return user.Features, err
		} else {
			warnf("AllFeatures called before client initialized")
			return map[string]Feature{}, nil
		}

	}

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

/*
DVCClientService Get variable by key for user data using Protobuf encoding

  - @param body

  - @param key Variable key

    -@return Variable
*/
func (c *DVCClient) Variable(userdata DVCUser, key string, defaultValue interface{}) (result Variable, err error) {
	if key == "" {
		return Variable{}, errors.New("invalid key provided for call to Variable")
	}

	convertedDefaultValue := convertDefaultValueType(defaultValue)
	variableType, err := variableTypeFromValue(key, convertedDefaultValue)

	if err != nil {
		return Variable{}, err
	}

	baseVar := BaseVariable{Key: key, Value: convertedDefaultValue, Type_: variableType}
	variable := Variable{BaseVariable: baseVar, DefaultValue: convertedDefaultValue, IsDefaulted: true}

	defer func() {
		if r := recover(); r != nil {
			// Return a usable default value in a panic situation
			result = variable
			err = fmt.Errorf("recovered from panic in Variable eval: %v - %s", r, string(debug.Stack()))
		}
	}()

	if !c.DevCycleOptions.EnableCloudBucketing {
		if !c.hasConfig() {
			warnf("Variable called before client initialized, returning default value")
			err = c.queueAggregateEvent(BucketedUserConfig{VariableVariationMap: map[string]FeatureVariation{}}, DVCEvent{
				Type_:  api.EventType_AggVariableDefaulted,
				Target: key,
			})

			if err != nil {
				warnf("Error queuing aggregate event: ", err)
			}

			return variable, nil
		}
		bucketedVariable, err := c.localBucketing.Variable(userdata, key, variableType)

		sameTypeAsDefault := compareTypes(bucketedVariable.Value, convertedDefaultValue)
		if bucketedVariable.Value != nil && sameTypeAsDefault {
			variable.Value = bucketedVariable.Value
			variable.IsDefaulted = false
		} else {
			if !sameTypeAsDefault && bucketedVariable.Value != nil {
				warnf("Type mismatch for variable %s. Expected type %s, got %s",
					key,
					reflect.TypeOf(defaultValue).String(),
					reflect.TypeOf(bucketedVariable.Value).String(),
				)
			}
		}
		return variable, err
	}

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
				warnf("Type mismatch for variable %s. Expected type %s, got %s",
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
		warnf("Error decoding response body %s", err)
		return variable, nil
	}
	warnf(v.Message)
	return variable, nil
}

func (c *DVCClient) AllVariables(user DVCUser) (map[string]ReadOnlyVariable, error) {
	var (
		httpMethod          = strings.ToUpper("Post")
		postBody            interface{}
		localVarReturnValue map[string]ReadOnlyVariable
	)
	if !c.DevCycleOptions.EnableCloudBucketing {
		if c.hasConfig() {
			user, err := c.generateBucketedConfig(user)
			if err != nil {
				return localVarReturnValue, err
			}
			return user.Variables, err
		} else {
			warnf("AllFeatures called before client initialized")
			return map[string]ReadOnlyVariable{}, nil
		}
	}

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

/*
DVCClientService Post events to DevCycle for user
  - @param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
  - @param body

@return InlineResponse201
*/

func (c *DVCClient) Track(user DVCUser, event DVCEvent) (bool, error) {
	if c.DevCycleOptions.DisableCustomEventLogging {
		return true, nil
	}
	if event.Type_ == "" {
		return false, errors.New("event type is required")
	}

	if !c.DevCycleOptions.EnableCloudBucketing {
		if c.hasConfig() {
			err := c.eventQueue.QueueEvent(user, event)
			return err == nil, err
		} else {
			warnf("Track called before client initialized")
			return true, nil
		}
	}
	var (
		httpMethod = strings.ToUpper("Post")
		postBody   interface{}
	)

	populatedUser := user.GetPopulatedUser(c.platformData)

	events := []DVCEvent{event}
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

func (c *DVCClient) FlushEvents() error {
	if c.DevCycleOptions.EnableCloudBucketing || !c.isInitialized {
		return nil
	}

	if c.DevCycleOptions.DisableCustomEventLogging && c.DevCycleOptions.DisableAutomaticEventLogging {
		return nil
	}

	err := c.eventQueue.FlushEvents()
	return err
}

func (c *DVCClient) SetClientCustomData(customData map[string]interface{}) error {
	if !c.DevCycleOptions.EnableCloudBucketing {
		if c.isInitialized {
			return c.localBucketing.SetClientCustomData(customData)
		} else {
			warnf("SetClientCustomData called before client initialized")
			return nil
		}
	}

	return errors.New("SetClientCustomData is not available in cloud bucketing mode")
}

/*
Close the client and flush any pending events. Stop any ongoing tickers
*/
func (c *DVCClient) Close() (err error) {
	if c.DevCycleOptions.EnableCloudBucketing {
		return
	}

	if !c.isInitialized {
		infof("Awaiting client initialization before closing")
		<-c.internalOnInitializedChannel
	}

	if c.eventQueue != nil {
		err = c.eventQueue.Close()
	}

	if c.configManager != nil {
		c.configManager.Close()
	}

	c.localBucketing.Close()

	return err
}

func (c *DVCClient) hasConfig() bool {
	return c.configManager.HasConfig()
}

func (c *DVCClient) performRequest(
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

func (c *DVCClient) handleError(r *http.Response, body []byte) (err error) {
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
		warnf("Server reported a 5xx error: ", newErr)
		return nil
	}
	return newErr
}

func compareTypes(value1 interface{}, value2 interface{}) bool {
	return reflect.TypeOf(value1) == reflect.TypeOf(value2)
}

func convertDefaultValueType(value interface{}) interface{} {
	switch value := value.(type) {
	case int:
		return float64(value)
	case int8:
		return float64(value)
	case int16:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case uint:
		return float64(value)
	case uint8:
		return float64(value)
	case uint16:
		return float64(value)
	case uint32:
		return float64(value)
	case uint64:
		return float64(value)
	case float32:
		return float64(value)
	default:
		return value
	}
}

func variableTypeFromValue(key string, value interface{}) (varType string, err error) {
	switch value.(type) {
	case float64:
		return "Number", nil
	case string:
		return "String", nil
	case bool:
		return "Boolean", nil
	case map[string]any:
		return "JSON", nil
	}

	return "", errorf("the default value for variable %s is not of type Boolean, Number, String, or JSON", key)
}

// callAPI do the request.
func (c *DVCClient) callAPI(request *http.Request) (*http.Response, error) {
	return c.cfg.HTTPClient.Do(request)
}

func exponentialBackoff(attempt int) float64 {
	delay := math.Pow(2, float64(attempt)) * 100
	randomSum := delay * 0.2 * rand.Float64()
	return (delay + randomSum)
}

// Change base path to allow switching to mocks
func (c *DVCClient) ChangeBasePath(path string) {
	c.cfg.BasePath = path
}

func (c *DVCClient) SetOptions(dvcOptions DVCOptions) {
	c.DevCycleOptions = &dvcOptions
}

// prepareRequest build the request
func (c *DVCClient) prepareRequest(
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

func sdkKeyIsValid(key string) bool {
	return strings.HasPrefix(key, "server") || strings.HasPrefix(key, "dvc_server")
}

// GenericError Provides access to the body, error and model on returned errors.
type GenericError struct {
	body  []byte
	error string
	model interface{}
}

// Error returns non-empty string if there was an error.
func (e GenericError) Error() string {
	return e.error
}

// Body returns the raw bytes of the response
func (e GenericError) Body() []byte {
	return e.body
}

// Model returns the unpacked model of the error
func (e GenericError) Model() interface{} {
	return e.model
}
