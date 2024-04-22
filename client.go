package devcycle

import (
	"context"
	"errors"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/devcyclehq/go-server-sdk/v2/variable-utils"
	"os"
	"regexp"
	"runtime"
	"strings"
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

// DevCycle Client
// In most cases there should be only one, shared, Client.
type Client struct {
	cfg             *HTTPConfiguration
	ctx             context.Context
	common          service // Reuse a single struct instead of allocating one for each service on the heap.
	DevCycleOptions *Options
	sdkKey          string
	cloudClient     *cloudClient
	configManager   *EnvironmentConfigManager
	eventQueue      *EventManager
	localBucketing  *NativeLocalBucketing
	platformData    *PlatformData
	// Set to true when the client has been initialized, regardless of whether the config has loaded successfully.
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
	client *Client
}

// NewClient creates a new API client.
// optionally pass a custom http.Client to allow for advanced features such as caching.
func NewClient(sdkKey string, options *Options) (*Client, error) {
	if sdkKey == "" {
		err := errors.New("missing sdk key! Call NewClient with a valid sdk key")
		util.Errorf("%v", err)
		return nil, err
	}
	if !sdkKeyIsValid(sdkKey) {
		return nil, fmt.Errorf("Invalid sdk key. Call NewClient with a valid sdk key.")
	}
	options.CheckDefaults()
	cfg := NewConfiguration(options)
	c := &Client{sdkKey: sdkKey}
	c.cfg = cfg
	c.ctx = context.Background()
	c.common.client = c
	c.DevCycleOptions = options

	if options.AdvancedOptions.OverridePlatformData != nil {
		c.platformData = options.AdvancedOptions.OverridePlatformData
	} else {
		c.platformData = GeneratePlatformData()
	}

	c.cloudClient = newCloudClient(sdkKey, options, c.platformData)

	if c.DevCycleOptions.Logger != nil {
		util.SetLogger(c.DevCycleOptions.Logger)
	}
	if c.IsLocalBucketing() {
		util.Infof("Using Native Bucketing")

		c.internalOnInitializedChannel = make(chan bool, 1)

		err := c.setLBClient(sdkKey, options)
		if err != nil {
			return c, fmt.Errorf("Error setting up local bucketing: %w", err)
		}

		c.eventQueue, err = NewEventManager(options, c.localBucketing, c.cfg, sdkKey)

		if err != nil {
			return c, fmt.Errorf("Error initializing event queue: %w", err)
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
	} else {
		util.Infof("Using Cloud Bucketing")
		if c.DevCycleOptions.OnInitializedChannel != nil {
			go func() {
				c.DevCycleOptions.OnInitializedChannel <- true
			}()
		}
	}
	return c, nil
}

func (c *Client) setLBClient(sdkKey string, options *Options) error {
	localBucketing, err := NewNativeLocalBucketing(sdkKey, c.platformData, options)
	if err != nil {
		return err
	}
	c.localBucketing = localBucketing

	return nil
}

func (c *Client) IsLocalBucketing() bool {
	return !c.DevCycleOptions.EnableCloudBucketing
}

func (c *Client) handleInitialization() {
	c.isInitialized = true

	if c.IsLocalBucketing() {
		util.Infof("Client initialized with local bucketing %v", c.localBucketing.GetClientUUID())
	}
	if c.DevCycleOptions.OnInitializedChannel != nil {
		go func() {
			c.DevCycleOptions.OnInitializedChannel <- true
		}()

	}
	c.internalOnInitializedChannel <- true
}

func (c *Client) generateBucketedConfig(user User) (config *BucketedUserConfig, err error) {
	config, err = c.localBucketing.GenerateBucketedConfigForUser(user)
	if err != nil {
		return nil, err
	}
	config.User = &user
	return
}

func (c *Client) GetRawConfig() (config []byte, etag string, err error) {
	if c.configManager == nil {
		return nil, "", errors.New("cannot read raw config; config manager is nil")
	}
	if c.configManager.HasConfig() {
		return c.configManager.GetRawConfig(), c.configManager.GetETag(), nil
	}
	return nil, "", errors.New("cannot read raw config; config manager has no config")
}

/*
Get all features by key for user data
  - @param body

@return map[string]Feature
*/
func (c *Client) AllFeatures(user User) (map[string]Feature, error) {
	if !c.IsLocalBucketing() {
		return c.cloudClient.AllFeatures(user)
	}

	if c.hasConfig() {
		user, err := c.generateBucketedConfig(user)
		if err != nil {
			return nil, fmt.Errorf("error generating bucketed config: %w", err)
		}
		return user.Features, err
	} else {
		util.Warnf("AllFeatures called before client initialized")
		return map[string]Feature{}, nil
	}
}

/*
VariableValue - Get variable value by key for user data

  - @param body

  - @param key Variable key

  - @param defaultValue Default value

    -@return interface{}
*/
func (c *Client) VariableValue(userdata User, key string, defaultValue interface{}) (interface{}, error) {
	variable, err := c.Variable(userdata, key, defaultValue)
	return variable.Value, err
}

/*
Variable - Get variable by key for user data

  - @param body

  - @param key Variable key

  - @param defaultValue Default value

    -@return Variable
*/
func (c *Client) Variable(userdata User, key string, defaultValue interface{}) (result Variable, err error) {
	if key == "" {
		return Variable{}, errors.New("invalid key provided for call to Variable")
	}

	convertedDefaultValue := variable_utils.ConvertDefaultValueType(defaultValue)
	variableType, err := variable_utils.VariableTypeFromValue(key, convertedDefaultValue, c.IsLocalBucketing())

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

	if !c.IsLocalBucketing() {
		return c.cloudClient.Variable(userdata, key, defaultValue)
	}

	return c.localBucketing.Variable(userdata, key, convertedDefaultValue)
}

func (c *Client) AllVariables(user User) (map[string]ReadOnlyVariable, error) {
	var (
		localVarReturnValue map[string]ReadOnlyVariable
	)
	if !c.IsLocalBucketing() {
		return c.cloudClient.AllVariables(user)
	}

	if c.hasConfig() {
		user, err := c.generateBucketedConfig(user)
		if err != nil {
			return localVarReturnValue, err
		}
		return user.Variables, err
	} else {
		util.Warnf("AllFeatures called before client initialized")
		return map[string]ReadOnlyVariable{}, nil
	}
}

/*
Post events to DevCycle for user
  - @param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
  - @param body

@return InlineResponse201
*/

func (c *Client) Track(user User, event Event) (bool, error) {
	if c.DevCycleOptions.DisableCustomEventLogging {
		return true, nil
	}
	if event.Type_ == "" {
		return false, errors.New("event type is required")
	}

	if !c.IsLocalBucketing() {
		return c.cloudClient.Track(user, event)
	}

	if c.hasConfig() {
		err := c.eventQueue.QueueEvent(user, event)
		if err != nil {
			util.Errorf("Error queuing event: %v", err)
			return false, err
		}
		return true, nil
	} else {
		util.Warnf("Track called before client initialized")
		return true, nil
	}
}

func (c *Client) FlushEvents() error {
	if !c.IsLocalBucketing() || !c.isInitialized {
		return nil
	}

	if c.DevCycleOptions.DisableCustomEventLogging && c.DevCycleOptions.DisableAutomaticEventLogging {
		return nil
	}

	err := c.eventQueue.FlushEvents()
	if err != nil {
		util.Errorf("Error flushing events: %v", err)
	}
	return err
}

func (c *Client) SetClientCustomData(customData map[string]interface{}) error {
	if c.IsLocalBucketing() {
		if c.isInitialized {
			return c.localBucketing.SetClientCustomData(customData)
		} else {
			util.Warnf("SetClientCustomData called before client initialized")
			return nil
		}
	}

	return errors.New("SetClientCustomData is not available in cloud bucketing mode")
}

/*
Close the client and flush any pending events. Stop any ongoing tickers
*/
func (c *Client) Close() (err error) {
	if !c.IsLocalBucketing() {
		return
	}

	if !c.isInitialized {
		util.Infof("Awaiting client initialization before closing")
		<-c.internalOnInitializedChannel
	}

	if c.eventQueue != nil {
		err = c.eventQueue.Close()
		if err != nil {
			util.Errorf("Error closing event queue: %v", err)
		}
	}

	if c.configManager != nil {
		c.configManager.Close()
	}

	c.localBucketing.Close()

	return err
}

func (c *Client) EventQueueMetrics() (int32, int32, int32) {
	return c.eventQueue.Metrics()
}

func (c *Client) hasConfig() bool {
	return c.configManager.HasConfig()
}

// Change base path to allow switching to mocks
func (c *Client) ChangeBasePath(path string) {
	c.cfg.BasePath = path
	c.cloudClient.ChangeBasePath(path)
}

func (c *Client) SetOptions(dvcOptions Options) {
	c.DevCycleOptions = &dvcOptions
	c.cloudClient.SetOptions(dvcOptions)
}

func sdkKeyIsValid(key string) bool {
	return strings.HasPrefix(key, "server") || strings.HasPrefix(key, "dvc_server")
}
