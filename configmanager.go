package devcycle

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/util"
)

const CONFIG_RETRIES = 1

type ConfigReceiver interface {
	StoreConfig([]byte, string, string, string) error
	GetRawConfig() []byte
	GetETag() string
	GetLastModified() string
	HasConfig() bool
}

type EnvironmentConfigManager struct {
	sdkKey               string
	minimalConfig        *api.MinimalConfig
	localBucketing       ConfigReceiver
	firstLoad            bool
	context              context.Context
	shutdown             context.CancelFunc
	pollingManager       *configPollingManager
	httpClient           *http.Client
	cfg                  *HTTPConfiguration
	sseManager           *SSEManager
	options              *Options
	InternalClientEvents chan api.ClientEvent
	eventManager         *EventManager
	pollingMutex         sync.Mutex
}

type configPollingManager struct {
	context     context.Context
	ticker      *time.Ticker
	stopPolling context.CancelFunc
}

func NewEnvironmentConfigManager(
	sdkKey string,
	localBucketing ConfigReceiver,
	manager *EventManager,
	options *Options,
	cfg *HTTPConfiguration,
) (configManager *EnvironmentConfigManager, err error) {
	configManager = &EnvironmentConfigManager{
		options:        options,
		sdkKey:         sdkKey,
		localBucketing: localBucketing,
		cfg:            cfg,
		httpClient:     cfg.HTTPClient,
		firstLoad:      true,
	}
	configManager.InternalClientEvents = make(chan api.ClientEvent, 100)

	configManager.context, configManager.shutdown = context.WithCancel(context.Background())
	configManager.eventManager = manager

	if options.EnableBetaRealtimeUpdates {
		sseManager, err := newSSEManager(configManager, options, cfg)
		if err != nil {
			return nil, err
		}
		configManager.sseManager = sseManager
		go configManager.ssePollingManager()
	} else {
		configManager.StartPolling(options.ConfigPollingIntervalMS)
	}
	return configManager, err
}

func (e *EnvironmentConfigManager) ssePollingManager() {
	for {
		select {
		case <-e.context.Done():
			util.Warnf("Stopping SSE polling.")
			return
		case event := <-e.InternalClientEvents:
			switch event.EventType {
			case api.ClientEventType_InternalNewConfigAvailable:
				minimumLastUpdated := event.EventData.(time.Time)
				if e.GetLastModified() != "" {
					currentLastModified, err := time.Parse(time.RFC1123, e.GetLastModified())
					if err != nil {
						util.Warnf("Error parsing last modified time: %s\n", err)
						e.InternalClientEvents <- api.ClientEvent{
							EventType: api.ClientEventType_Error,
							EventData: "Error parsing last modified time: " + err.Error(),
							Status:    "error",
							Error:     err,
						}
					}
					if currentLastModified.After(minimumLastUpdated) {
						// Skip fetching config if the current config is newer than the minimumLastUpdated
						continue
					}
				}

				err := e.fetchConfig(CONFIG_RETRIES, minimumLastUpdated)
				if err != nil {
					util.Warnf("Error fetching config: %s\n", err)
					e.InternalClientEvents <- api.ClientEvent{
						EventType: api.ClientEventType_Error,
						EventData: "Error fetching config: " + err.Error(),
						Status:    "error",
						Error:     err,
					}
				}

			case api.ClientEventType_InternalSSEFailure:
				// Re-enable polling until a valid config is fetched, and then re-initialize SSE.
				e.sseManager.StopSSE()
				e.StartPolling(e.options.ConfigPollingIntervalMS)

			case api.ClientEventType_InternalSSEConnected:
				e.StartPolling(time.Minute * 10)

			case api.ClientEventType_ConfigUpdated:
				eventData := event.EventData.(map[string]string)

				if url, ok := eventData["sseUrl"]; ok && e.options.EnableBetaRealtimeUpdates && e.sseManager != nil {
					// Reconnect SSE
					if url != "" && (e.sseManager.url != url || !e.sseManager.Connected.Load()) {
						err := e.StartSSE(url)
						if err != nil {
							e.InternalClientEvents <- api.ClientEvent{
								EventType: api.ClientEventType_Error,
								EventData: "Error starting SSE after config update: " + err.Error(),
								Status:    "error",
								Error:     err,
							}
						}
					}
				}
			}
		}
	}
}

func (e *EnvironmentConfigManager) StartSSE(url string) error {
	if !e.options.EnableBetaRealtimeUpdates {
		return fmt.Errorf("realtime updates are disabled. Cannot start SSE")
	}
	return e.sseManager.StartSSEOverride(url)
}

func (e *EnvironmentConfigManager) StopPolling() {
	if e.pollingManager != nil {
		e.pollingManager.stopPolling()
	}
}

func (e *EnvironmentConfigManager) StartPolling(interval time.Duration) {
	e.pollingMutex.Lock()
	defer e.pollingMutex.Unlock()
	if e.pollingManager != nil {
		e.pollingManager.stopPolling()
	}
	pollingManager := &configPollingManager{
		context:     nil,
		ticker:      time.NewTicker(interval),
		stopPolling: nil,
	}
	pollingManager.context, pollingManager.stopPolling = context.WithCancel(e.context)

	e.pollingManager = pollingManager
	go func() {
		for {
			if e.pollingManager == nil {
				return
			}
			select {
			case <-e.context.Done():
				util.Warnf("Stopping config polling.")
				e.pollingManager.ticker.Stop()
				return
			case <-e.pollingManager.ticker.C:
				err := e.fetchConfig(CONFIG_RETRIES)
				if err != nil {
					util.Warnf("Error fetching config: %s\n", err)
				}
			}
		}
	}()
}

func (e *EnvironmentConfigManager) initialFetch() error {

	return e.fetchConfig(CONFIG_RETRIES)
}

func (e *EnvironmentConfigManager) fetchConfig(numRetriesRemaining int, minimumLastModified ...time.Time) (err error) {
	if numRetriesRemaining < 0 {
		return fmt.Errorf("retries exhausted")
	}
	util.Debugf("Fetching config. Retries remaining: %d\n", numRetriesRemaining)
	defer func() {
		if r := recover(); r != nil {
			// get the stack trace and potentially log it here
			err = fmt.Errorf("recovered from panic in fetchConfig: %v", r)
		}
	}()

	req, err := http.NewRequest("GET", e.getConfigURL(), nil)
	if err != nil {
		return err
	}

	etag := e.localBucketing.GetETag()
	lastModified := e.localBucketing.GetLastModified()
	storedLM, err := time.Parse(time.RFC1123, lastModified)
	if err != nil {
		util.Warnf("Error parsing last modified time: %s\n", err)
	}
	if len(minimumLastModified) > 0 && storedLM.Before(minimumLastModified[0]) {
		lastModified = minimumLastModified[0].Format(time.RFC1123)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}
	if etag != "" && !e.options.DisableETagMatching {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := e.httpClient.Do(req)

	if err != nil {
		if numRetriesRemaining > 0 {
			util.Warnf("Retrying config fetch %d more times. Error: %s", numRetriesRemaining, err)
			return e.fetchConfig(numRetriesRemaining - 1)
		}
		return err
	}
	lastModifiedHeader := resp.Header.Get("Last-Modified")
	if lastModifiedHeader != "" {
		responseLastModified, parseError := time.Parse(time.RFC1123, lastModifiedHeader)
		if parseError == nil {
			if storedLM.After(responseLastModified) {
				return e.fetchConfig(numRetriesRemaining - 1)
			}
			if len(minimumLastModified) > 0 && responseLastModified.Before(minimumLastModified[0]) && numRetriesRemaining > 0 {
				return e.fetchConfig(numRetriesRemaining-1, minimumLastModified...)
			}
		}
	}

	defer resp.Body.Close()

	switch statusCode := resp.StatusCode; {
	case statusCode == http.StatusOK:
		resp.Request = req
		return e.setConfigFromResponse(resp)
	case statusCode == http.StatusNotModified:
		if e.sseManager != nil && !e.sseManager.Connected.Load() && e.minimalConfig != nil && e.minimalConfig.SSE != nil {
			configUpdatedEvent := api.ClientEvent{
				EventType: api.ClientEventType_ConfigUpdated,
				EventData: map[string]string{
					"rayId":        resp.Header.Get("Cf-Ray"),
					"eTag":         resp.Header.Get("Etag"),
					"lastModified": lastModified,
					"sseUrl":       fmt.Sprintf("%s%s", e.minimalConfig.SSE.Hostname, e.minimalConfig.SSE.Path),
				},
				Status: "success",
				Error:  nil,
			}
			e.InternalClientEvents <- configUpdatedEvent
		}
		return nil
	case statusCode == http.StatusForbidden:
		e.StopPolling()
		return fmt.Errorf("invalid SDK key. Aborting config polling")
	case statusCode >= 500:
		// Retryable Errors. Continue polling.
		util.Warnf("Config fetch failed. Status:" + resp.Status)
	default:
		err = fmt.Errorf("Unexpected response code: %d\n"+
			"Body: %s\n"+
			"URL: %s\n"+
			"Headers: %s\n"+
			"Could not download configuration. Using cached version if available %s\n",
			resp.StatusCode, resp.Body, e.getConfigURL(), resp.Header, resp.Header.Get("ETag"))
	}

	if numRetriesRemaining > 0 {
		util.Warnf("Retrying config fetch %d more times. Status: %s", numRetriesRemaining, resp.Status)
		return e.fetchConfig(numRetriesRemaining - 1)
	}

	return err
}

func (e *EnvironmentConfigManager) setConfigFromResponse(response *http.Response) error {
	config, err := io.ReadAll(response.Body)

	if err != nil {
		return err
	}
	// Check
	valid := json.Valid(config)
	if !valid {
		return fmt.Errorf("invalid JSON data received for config")
	}

	err = e.setConfig(
		config,
		response.Header.Get("Etag"),
		response.Header.Get("Cf-Ray"),
		response.Header.Get("Last-Modified"),
	)

	if err != nil {
		return err
	}

	util.Infof("Config set. ETag: %s Last-Modified: %s\n", e.localBucketing.GetETag(), e.localBucketing.GetLastModified())
	if e.eventManager != nil {
		err = e.eventManager.QueueSDKConfigEvent(*response.Request, *response)
		if err != nil {
			util.Warnf("Error queuing SDK config event: %s\n", err)
		}
	}

	if e.firstLoad {
		e.firstLoad = false
		util.Infof("DevCycle SDK Initialized.")
	}
	return nil
}

func (e *EnvironmentConfigManager) setConfig(config []byte, eTag, rayId, lastModified string) error {
	configUpdatedEvent := api.ClientEvent{
		EventType: api.ClientEventType_ConfigUpdated,
		EventData: map[string]string{
			"rayId":        rayId,
			"eTag":         eTag,
			"lastModified": lastModified,
			"sseUrl":       "",
		},
		Status: "success",
		Error:  nil,
	}
	defer func() {
		go func() {
			e.InternalClientEvents <- configUpdatedEvent
			if e.options.ClientEventHandler != nil {
				e.options.ClientEventHandler <- configUpdatedEvent
			}
		}()
	}()
	err := e.localBucketing.StoreConfig(config, eTag, rayId, lastModified)
	if err != nil {
		configUpdatedEvent.EventType = api.ClientEventType_Error
		configUpdatedEvent.Status = "error"
		configUpdatedEvent.Error = err
		return err
	}

	err = json.Unmarshal(e.GetRawConfig(), &e.minimalConfig)
	if err != nil {
		configUpdatedEvent.EventType = api.ClientEventType_Error
		configUpdatedEvent.Status = "error"
		configUpdatedEvent.Error = err
		return err
	}
	if e.minimalConfig != nil && e.minimalConfig.SSE != nil {
		sseUrl := fmt.Sprintf("%s%s", e.minimalConfig.SSE.Hostname, e.minimalConfig.SSE.Path)
		if e.sseManager != nil {
			configUpdatedEvent.EventData.(map[string]string)["sseUrl"] = sseUrl
		}
	}
	return nil
}

func (e *EnvironmentConfigManager) getConfigURL() string {
	configBasePath := e.cfg.ConfigCDNBasePath

	return fmt.Sprintf("%s/config/v1/server/%s.json", configBasePath, e.sdkKey)
}

func (e *EnvironmentConfigManager) HasConfig() bool {
	return e.localBucketing.HasConfig()
}

func (e *EnvironmentConfigManager) GetRawConfig() []byte {
	return e.localBucketing.GetRawConfig()
}

func (e *EnvironmentConfigManager) GetETag() string {
	return e.localBucketing.GetETag()
}

func (e *EnvironmentConfigManager) GetLastModified() string {
	return e.localBucketing.GetLastModified()
}

func (e *EnvironmentConfigManager) Close() {
	e.shutdown()
	e.pollingMutex.Lock()
	defer e.pollingMutex.Unlock()
	if e.pollingManager != nil {
		e.pollingManager.stopPolling()
	}
	if e.sseManager != nil {
		e.sseManager.Close()
	}
}
