package devcycle

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"io"
	"net/http"
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
	sdkKey         string
	minimalConfig  *api.MinimalConfig
	localBucketing ConfigReceiver
	firstLoad      bool
	context        context.Context
	stopPolling    context.CancelFunc
	httpClient     *http.Client
	cfg            *HTTPConfiguration
	ticker         *time.Ticker
	sseManager     *SSEManager
	options        *Options
}

func NewEnvironmentConfigManager(
	sdkKey string,
	localBucketing ConfigReceiver,
	options *Options,
	cfg *HTTPConfiguration,
) (e *EnvironmentConfigManager) {
	configManager := &EnvironmentConfigManager{
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

	configManager.sseManager = newSSEManager(configManager, options)

	configManager.context, configManager.stopPolling = context.WithCancel(context.Background())

	return configManager
}

func (e *EnvironmentConfigManager) StartSSE() error {
	err := e.initialFetch()
	if err != nil {
		return err
	}
	if e.options.AdvancedOptions.ServerSentEventsURI == "" {
		util.Warnf("Server Sent Events URI not set. Aborting SSE connection. Falling back to polling")
		e.StartPolling(e.options.ConfigPollingIntervalMS)
		return fmt.Errorf("server Sent Events URI not set. Aborting SSE connection. Falling back to polling")
	}
	return e.sseManager.StartSSE()
}

func (e *EnvironmentConfigManager) StartPolling(
	interval time.Duration,
) {
	if e.ticker == nil {
		e.ticker = time.NewTicker(interval)
	}

	go func() {
		for {
			select {
			case <-e.context.Done():
				util.Warnf("Stopping config polling.")
				e.ticker.Stop()
				return
			case <-e.ticker.C:
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

func (e *EnvironmentConfigManager) fetchConfig(numRetriesRemaining int) (err error) {
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
	switch statusCode := resp.StatusCode; {
	case statusCode == http.StatusOK:
		return e.setConfigFromResponse(resp)
	case statusCode == http.StatusNotModified:
		return nil
	case statusCode == http.StatusForbidden:
		e.stopPolling()
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
	err := e.localBucketing.StoreConfig(config, eTag, rayId, lastModified)
	if err != nil {
		return err
	}

	err = json.Unmarshal(e.GetRawConfig(), &e.minimalConfig)
	if err != nil {
		return err
	}
	if e.minimalConfig != nil && e.minimalConfig.SSE != nil {
		e.options.AdvancedOptions.ServerSentEventsURI = fmt.Sprintf("%s%s", e.minimalConfig.SSE.Hostname, e.minimalConfig.SSE.Path)
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
	e.stopPolling()
}
