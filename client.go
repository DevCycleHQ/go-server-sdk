package devcycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	cfg                          *HTTPConfiguration
	common                       service // Reuse a single struct instead of allocating one for each service on the heap.
	DevCycleOptions              *DVCOptions
	sdkKey                       string
	auth                         context.Context
	localBucketing               *DevCycleLocalBucketing
	configManager                *EnvironmentConfigManager
	eventQueue                   *EventQueue
	isInitialized                bool
	internalOnInitializedChannel chan bool
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

func initializeLocalBucketing(sdkKey string, options *DVCOptions) (ret *DevCycleLocalBucketing, err error) {
	options.CheckDefaults()
	ret = &DevCycleLocalBucketing{}
	err = ret.Initialize(sdkKey, options)
	if err != nil {
		errorf("error while initializing local bucketing", err)
		return nil, err
	}
	return
}

func setLBClient(sdkKey string, options *DVCOptions, c *DVCClient) error {
	localBucketing, err := initializeLocalBucketing(sdkKey, options)

	if err != nil {
		return err
	}

	c.eventQueue = &EventQueue{}
	err = c.eventQueue.initialize(options, localBucketing, c.cfg)

	if err != nil {
		return err
	}

	c.localBucketing = localBucketing
	c.configManager = &EnvironmentConfigManager{localBucketing: localBucketing}
	err = c.configManager.Initialize(sdkKey, localBucketing, c.cfg)

	if err != nil {
		return err
	}

	return err
}

// NewDVCClient creates a new API client.
// optionally pass a custom http.Client to allow for advanced features such as caching.
func NewDVCClient(sdkKey string, options *DVCOptions) (*DVCClient, error) {
	if sdkKey == "" {
		return nil, fmt.Errorf("missing sdk key! Call NewDVCClient with a valid sdk key")
	}
	if !sdkKeyIsValid(sdkKey) {
		return nil, fmt.Errorf("Invalid sdk key. Call NewDVCClient with a valid sdk key.")
	}
	cfg := NewConfiguration(options)

	options.CheckDefaults()

	c := &DVCClient{sdkKey: sdkKey}
	c.cfg = cfg
	c.common.client = c
	c.DevCycleOptions = options

	if c.DevCycleOptions.Logger != nil {
		SetLogger(c.DevCycleOptions.Logger)
	}

	c.internalOnInitializedChannel = make(chan bool, 1)
	err := setLBClient(sdkKey, options, c)
	if err != nil {
		return c, err
	}
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

func (c *DVCClient) generateBucketedConfig(user DVCUser) (config BucketedUserConfig, err error) {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return BucketedUserConfig{}, err
	}
	config, err = c.localBucketing.GenerateBucketedConfigForUser(string(userJSON))
	if err != nil {
		return BucketedUserConfig{}, err
	}
	config.user = &user
	return
}

func (c *DVCClient) variableForUser(user DVCUser, key string, variableType VariableTypeCode) (variable Variable, err error) {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return Variable{}, err
	}
	variable, err = c.localBucketing.VariableForUser(userJSON, key, variableType)
	return
}

func (c *DVCClient) queueEvent(user DVCUser, event DVCEvent) (err error) {
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
func (c *DVCClient) AllFeatures(user DVCUser) (map[string]Feature, error) {
	if c.hasConfig() {
		user, err := c.generateBucketedConfig(user)
		return user.Features, err
	} else {
		warnf("AllFeatures called before client initialized")
		return map[string]Feature{}, nil
	}
}

/*
DVCClientService Get variable by key for user data
  - @param body
  - @param key Variable key

@return Variable
*/
func (c *DVCClient) Variable(userdata DVCUser, key string, defaultValue interface{}) (Variable, error) {
	if key == "" {
		return Variable{}, errors.New("invalid key provided for call to Variable")
	}

	convertedDefaultValue := convertDefaultValueType(defaultValue)
	variableType, err := variableTypeFromValue(key, convertedDefaultValue)

	if err != nil {
		return Variable{}, err
	}

	baseVar := baseVariable{Key: key, Value: convertedDefaultValue, Type_: variableType}
	variable := Variable{baseVariable: baseVar, DefaultValue: convertedDefaultValue, IsDefaulted: true}

	if !c.hasConfig() {
		warnf("Variable called before client initialized, returning default value")
		err = c.queueAggregateEvent(BucketedUserConfig{VariableVariationMap: map[string]FeatureVariation{}}, DVCEvent{
			Type_:  EventType_AggVariableDefaulted,
			Target: key,
		})

		if err != nil {
			warnf("Error queuing aggregate event: ", err)
			err = nil
		}

		return variable, nil
	}
	variableTypeCode, err := c.variableTypeCodeFromType(variableType)

	if err != nil {
		return Variable{}, err
	}
	bucketedVariable, err := c.variableForUser(userdata, key, variableTypeCode)

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

func (c *DVCClient) AllVariables(user DVCUser) (map[string]ReadOnlyVariable, error) {
	var (
		localVarReturnValue map[string]ReadOnlyVariable
	)

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

	if c.isInitialized {
		err := c.eventQueue.QueueEvent(user, event)
		return err == nil, err
	} else {
		warnf("Track called before client initialized")
		return true, nil
	}
}

func (c *DVCClient) FlushEvents() error {
	if !c.isInitialized {
		return nil
	}

	if c.DevCycleOptions.DisableCustomEventLogging && c.DevCycleOptions.DisableAutomaticEventLogging {
		return nil
	}

	err := c.eventQueue.FlushEvents()
	return err
}

func (c *DVCClient) SetClientCustomData(customData map[string]interface{}) error {
	if c.isInitialized {
		data, err := json.Marshal(customData)
		if err != nil {
			return err
		}
		err = c.localBucketing.SetClientCustomData(string(data))
		return err
	} else {
		warnf("SetClientCustomData called before client initialized")
		return nil
	}
}

/*
Close the client and flush any pending events. Stop any ongoing tickers
*/
func (c *DVCClient) Close() (err error) {
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

	return err
}

func (c *DVCClient) hasConfig() bool {
	return c.configManager.hasConfig
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

func (c *DVCClient) variableTypeCodeFromType(varType string) (varTypeCode VariableTypeCode, err error) {
	switch varType {
	case "Boolean":
		return c.localBucketing.VariableTypeCodes.Boolean, nil
	case "Number":
		return c.localBucketing.VariableTypeCodes.Number, nil
	case "String":
		return c.localBucketing.VariableTypeCodes.String, nil
	case "JSON":
		return c.localBucketing.VariableTypeCodes.JSON, nil
	}

	return 0, fmt.Errorf("variable type %s is not a valid type", varType)
}

// Change base path to allow switching to mocks
func (c *DVCClient) ChangeBasePath(path string) {
	c.cfg.BasePath = path
}

func (c *DVCClient) SetOptions(dvcOptions DVCOptions) {
	c.DevCycleOptions = &dvcOptions
}

func sdkKeyIsValid(key string) bool {
	return strings.HasPrefix(key, "server") || strings.HasPrefix(key, "dvc_server")
}
