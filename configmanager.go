package devcycle

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"io"
	"net/http"
	"strings"
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
}

type configPollingManager struct {
	context     context.Context
	ticker      *time.Ticker
	stopPolling context.CancelFunc
}

func NewEnvironmentConfigManager(
	sdkKey string,
	localBucketing ConfigReceiver,
	options *Options,
	cfg *HTTPConfiguration,
) (configManager *EnvironmentConfigManager) {
	configManager = &EnvironmentConfigManager{
		options:        options,
		sdkKey:         sdkKey,
		localBucketing: localBucketing,
		cfg:            cfg,
		httpClient: &http.Client{
			// Set an explicit timeout so that we don't wait forever on a request
			// Use the configurable timeout because fetching the first config can block SDK initialization.
			Timeout: options.RequestTimeout,
		},
		firstLoad: true,
	}
	configManager.InternalClientEvents = make(chan api.ClientEvent, 100)
	configManager.sseManager = newSSEManager(configManager, options)

	configManager.context, configManager.shutdown = context.WithCancel(context.Background())
	if options.EnableRealtimeUpdates {
		go configManager.ssePollingManager()
	}
	return configManager
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
				break
			case api.ClientEventType_InternalSSEFailure:
				// Re-enable polling until a valid config is fetched, and then re-initialize SSE.
				e.sseManager.StopSSE()
				err := e.StartPolling(e.options.ConfigPollingIntervalMS)
				if err != nil {
					e.InternalClientEvents <- api.ClientEvent{
						EventType: api.ClientEventType_Error,
						EventData: "Error starting polling after SSE failure: " + err.Error(),
						Status:    "error",
						Error:     err,
					}
				}
				break
			case api.ClientEventType_InternalSSEConnected:
				if e.pollingManager != nil {
					e.pollingManager.stopPolling()
				}
				break
			case api.ClientEventType_ConfigUpdated:
				if strings.Contains(event.EventData.(string), "SSE URL") {
					// Reconnect SSE
					e.sseManager.StopSSE()
					err := e.sseManager.StartSSE()
					if err != nil {
						e.InternalClientEvents <- api.ClientEvent{
							EventType: api.ClientEventType_Error,
							EventData: "Error starting SSE after config update: " + err.Error(),
							Status:    "error",
							Error:     err,
						}
					}
				}
				break
			}
		}
	}
}

func (e *EnvironmentConfigManager) StartSSE() error {
	if !e.options.EnableRealtimeUpdates {
		return fmt.Errorf("realtime updates are disabled. Cannot start SSE")
	}

	if e.sseManager.URL == "" {
		util.Warnf("Server Sent Events URI not set. Aborting SSE connection. Falling back to polling")
		return fmt.Errorf("server Sent Events URI not set. Aborting SSE connection. Falling back to polling")
	}
	return e.sseManager.StartSSE()
}

func (e *EnvironmentConfigManager) GetSSE() *SSEManager {
	return e.sseManager
}

func (e *EnvironmentConfigManager) StartPolling(interval time.Duration) error {
	if e.pollingManager != nil {
		e.pollingManager.stopPolling()
	}
	pollingManager := &configPollingManager{
		context:     nil,
		ticker:      time.NewTicker(interval),
		stopPolling: nil,
	}
	pollingManager.context, pollingManager.stopPolling = context.WithCancel(context.Background())

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
	return nil
}

func (e *EnvironmentConfigManager) initialFetch() error {

	err := e.fetchConfig(CONFIG_RETRIES)
	if err != nil {
		return err
	}
	if e.options.ClientEventHandler != nil {
		go func() {
			e.options.ClientEventHandler <- api.ClientEvent{
				EventType: api.ClientEventType_Initialized,
				EventData: "Initialized DevCycle SDK.",
				Status:    "success",
				Error:     nil,
			}
		}()
	}
	return nil
}

func (e *EnvironmentConfigManager) fetchConfig(numRetriesRemaining int, minimumLastModified ...time.Time) (err error) {
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

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, err := e.httpClient.Do(req)

	if err != nil {
		if numRetriesRemaining > 0 {
			util.Warnf("Retrying config fetch %d more times. Error: %s", numRetriesRemaining, err)
			return e.fetchConfig(numRetriesRemaining - 1)
		}
		return err
	}
	defer resp.Body.Close()

	if len(minimumLastModified) > 0 {
		respLastModified := resp.Header.Get("Last-Modified")
		if respLastModified == "" {
			return e.fetchConfig(numRetriesRemaining, minimumLastModified...)
		}
		respLMTime, timeError := time.Parse(time.RFC1123, respLastModified)
		if timeError != nil {
			return e.fetchConfig(numRetriesRemaining, minimumLastModified...)
		}
		if respLMTime.Before(minimumLastModified[0]) {
			return e.fetchConfig(numRetriesRemaining, minimumLastModified...)
		}
	}

	switch statusCode := resp.StatusCode; {
	case statusCode == http.StatusOK:
		return e.setConfigFromResponse(resp)
	case statusCode == http.StatusNotModified:
		return nil
	case statusCode == http.StatusForbidden:
		e.pollingManager.stopPolling()
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

	if e.firstLoad {
		e.firstLoad = false
		util.Infof("DevCycle SDK Initialized.")
	}
	return nil
}

func (e *EnvironmentConfigManager) setConfig(config []byte, eTag, rayId, lastModified string) error {
	configUpdatedEvent := api.ClientEvent{
		EventType: api.ClientEventType_ConfigUpdated,
		EventData: fmt.Sprintf("Config updated. RayId: %s ETag: %s Last-Modified: %s", rayId, eTag, lastModified),
		Status:    "success",
		Error:     nil,
	}
	defer func() {
		go func() {
			e.InternalClientEvents <- configUpdatedEvent
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
		if e.sseManager.URL != sseUrl {
			e.sseManager.URL = sseUrl
			configUpdatedEvent.EventData = fmt.Sprintf("%s SSE URL: %s", configUpdatedEvent.EventData, e.sseManager.URL)
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
	if e.pollingManager != nil {
		e.pollingManager.stopPolling()
	}
	if e.sseManager != nil {
		e.sseManager.StopSSE()
	}

}
