package devcycle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

var (
	jsonCheck = regexp.MustCompile("(?i:[application|text]/json)")
	xmlCheck  = regexp.MustCompile("(?i:[application|text]/xml)")
)

// DVCClient
// In most cases there should be only one, shared, DVCClient.
type DVCClient struct {
	cfg             *HTTPConfiguration
	common          service // Reuse a single struct instead of allocating one for each service on the heap.
	DevCycleOptions *DVCOptions
	environmentKey  string
	auth            context.Context
	localBucketing  *DevCycleLocalBucketing
	configManager   *EnvironmentConfigManager
	eventQueue      *EventQueue
	isInitialized   bool
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

func initializeLocalBucketing(environmentKey string, options *DVCOptions) (ret *DevCycleLocalBucketing, err error) {
	cfg := NewConfiguration(options)

	options.CheckDefaults()
	ret = &DevCycleLocalBucketing{}
	err = ret.Initialize(environmentKey, options, cfg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return
}

func setLBClient(environmentKey string, options *DVCOptions, c *DVCClient) (*DVCClient, error) {
	localBucketing, err := initializeLocalBucketing(environmentKey, options)

	if err != nil {
		if options.OnInitializedChannel != nil {
			options.OnInitializedChannel <- true
		}
		return nil, err
	}
	c.localBucketing = localBucketing
	c.configManager = c.localBucketing.configManager
	c.eventQueue = c.localBucketing.eventQueue
	c.isInitialized = c.configManager.HasConfig()
	if options.OnInitializedChannel != nil {
		options.OnInitializedChannel <- true
		close(options.OnInitializedChannel)
	}
	return c, nil
}

// NewDVCClient creates a new API client.
// optionally pass a custom http.Client to allow for advanced features such as caching.
func NewDVCClient(environmentKey string, options *DVCOptions) (*DVCClient, error) {
	if environmentKey == "" {
		return nil, fmt.Errorf("Missing environment key! Call NewDVCClient with a valid environment key.")
	}
	if !environmentKeyIsValid(environmentKey) {
		return nil, fmt.Errorf("Invalid environment key. Call NewDVCClient with a valid environment key.")
	}
	cfg := NewConfiguration(options)

	options.CheckDefaults()

	c := &DVCClient{environmentKey: environmentKey}
	c.cfg = cfg
	c.common.client = c
	c.DevCycleOptions = options

	if !c.DevCycleOptions.EnableCloudBucketing {
		if c.DevCycleOptions.OnInitializedChannel != nil {
			go setLBClient(environmentKey, options, c)
		} else {
			return setLBClient(environmentKey, options, c)
		}
	}
	return c, nil
}

func (c *DVCClient) generateBucketedConfig(body dvcPopulatedUser) (user BucketedUserConfig, err error) {
	userJSON, err := json.Marshal(body)
	if err != nil {
		return BucketedUserConfig{}, err
	}
	user, err = c.localBucketing.GenerateBucketedConfigForUser(string(userJSON))
	if err != nil {
		return BucketedUserConfig{}, err
	}
	user.user = &body
	return
}

func (c *DVCClient) queueEvent(user dvcPopulatedUser, event DVCEvent) (err error) {
	err = c.eventQueue.QueueEvent(user, event)
	return
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
func (c *DVCClient) AllFeatures(body DVCUser) (map[string]Feature, error) {
	populatedUser := body.getPopulatedUser()
	if !c.DevCycleOptions.EnableCloudBucketing {
		if c.isInitialized {
			user, err := c.generateBucketedConfig(populatedUser)
			return user.Features, err
		} else {
			log.Println("AllFeatures called before client initialized")
			return map[string]Feature{}, nil
		}

	}
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
DVCClientService Get variable by key for user data
  - @param body
  - @param key Variable key

@return Variable
*/
func (c *DVCClient) Variable(userdata DVCUser, key string, defaultValue interface{}) (Variable, error) {
	populatedUser := userdata.getPopulatedUser()
	convertedDefaultValue := convertDefaultValueType(defaultValue)
	variableType, err := variableTypeFromValue(key, convertedDefaultValue)

	if err != nil {
		return Variable{}, err
	}

	baseVar := baseVariable{Key: key, Value: convertedDefaultValue, Type_: variableType}
	variable := Variable{baseVariable: baseVar, DefaultValue: convertedDefaultValue, IsDefaulted: true}

	if !c.DevCycleOptions.EnableCloudBucketing {
		if !c.isInitialized {
			log.Println("Variable called before client initialized, returning default value")
			return variable, nil
		}
		bucketed, err := c.generateBucketedConfig(populatedUser)

		sameTypeAsDefault := compareTypes(bucketed.Variables[key].Value, convertedDefaultValue)
		variableEvaluationType := ""
		if bucketed.Variables[key].Value != nil && sameTypeAsDefault {
			variable.Value = bucketed.Variables[key].Value
			variable.IsDefaulted = false
			variableEvaluationType = EventType_AggVariableEvaluated
		} else {
			if !sameTypeAsDefault && bucketed.Variables[key].Value != nil {
				log.Printf("Type mismatch for variable %s. Expected type %s, got %s", key, reflect.TypeOf(defaultValue).String(), reflect.TypeOf(bucketed.Variables[key].Value).String())
			}
			variableEvaluationType = EventType_AggVariableDefaulted
		}
		if !c.DevCycleOptions.DisableAutomaticEventLogging {
			err = c.queueAggregateEvent(bucketed, DVCEvent{
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
	path := c.cfg.BasePath + "/v1/variables/{key}"
	path = strings.Replace(path, "{"+"key"+"}", fmt.Sprintf("%v", key), -1)

	headers := make(map[string]string)
	queryParams := url.Values{}

	// userdata params
	postBody = &populatedUser

	r, body, err := c.performRequest(path, httpMethod, postBody, headers, queryParams)

	if err != nil {
		return localVarReturnValue, err
	}

	if r.StatusCode < 300 {
		// If we succeed, return the data, otherwise pass on to decode error.
		err = decode(&localVarReturnValue, body, r.Header.Get("Content-Type"))
		if err == nil {
			return localVarReturnValue, err
		}
	}

	var v ErrorResponse
	err = decode(&v, body, r.Header.Get("Content-Type"))
	if err != nil {
		log.Println(err.Error())
		return variable, nil
	}
	log.Println(v.Message)
	return variable, nil
}

func (c *DVCClient) AllVariables(body DVCUser) (map[string]ReadOnlyVariable, error) {
	populatedUser := body.getPopulatedUser()
	var (
		httpMethod          = strings.ToUpper("Post")
		postBody            interface{}
		localVarReturnValue map[string]ReadOnlyVariable
	)
	if !c.DevCycleOptions.EnableCloudBucketing {
		if c.isInitialized {
			user, err := c.generateBucketedConfig(populatedUser)
			if err != nil {
				return localVarReturnValue, err
			}
			return user.Variables, err
		} else {
			log.Println("AllFeatures called before client initialized")
			return map[string]ReadOnlyVariable{}, nil
		}
	}

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
	populatedUser := user.getPopulatedUser()
	if c.DevCycleOptions.DisableCustomEventLogging {
		return true, nil
	}
	if event.Type_ == "" {
		return false, errors.New("event type is required")
	}

	if !c.DevCycleOptions.EnableCloudBucketing {
		if c.isInitialized {
			err := c.eventQueue.QueueEvent(populatedUser, event)
			return err == nil, err
		} else {
			log.Println("Track called before client initialized")
			return true, nil
		}
	}
	var (
		httpMethod = strings.ToUpper("Post")
		postBody   interface{}
	)

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

/*
Close the client and flush any pending events. Stop any ongoing tickers
*/
func (c *DVCClient) Close() (err error) {
	if c.DevCycleOptions.EnableCloudBucketing || !c.isInitialized {
		return
	}

	err = c.eventQueue.Close()
	c.configManager.Close()
	return err
}

func (c *DVCClient) performRequest(
	path string, method string,
	postBody interface{},
	headerParams map[string]string,
	queryParams url.Values,
) (response *http.Response, body []byte, err error) {
	headerParams["Content-Type"] = "application/json"
	headerParams["Accept"] = "application/json"
	headerParams["Authorization"] = c.environmentKey

	r, err := c.prepareRequest(
		path,
		method,
		postBody,
		headerParams,
		queryParams,
	)

	if err != nil {
		return nil, nil, err
	}

	httpResponse, err := c.callAPI(r)
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
		log.Println("Request error: ", newErr)
		return nil
	}
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

	return "", fmt.Errorf("the default value for variable %s is not of type Boolean, Number, String, or JSON", key)
}

// callAPI do the request.
func (c *DVCClient) callAPI(request *http.Request) (*http.Response, error) {
	return c.cfg.HTTPClient.Do(request)
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
	url, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	// Adding Query Param
	query := url.Query()
	for k, v := range queryParams {
		for _, iv := range v {
			query.Add(k, iv)
		}
	}

	if c.DevCycleOptions.EnableEdgeDB {
		query.Add("enableEdgeDB", "true")
	}

	// Encode the parameters.
	url.RawQuery = query.Encode()

	// Generate a new request
	if body != nil {
		localVarRequest, err = http.NewRequest(method, url.String(), body)
	} else {
		localVarRequest, err = http.NewRequest(method, url.String(), nil)
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

func environmentKeyIsValid(key string) bool {
	return strings.HasPrefix(key, "server") || strings.HasPrefix(key, "dvc_server")
}
