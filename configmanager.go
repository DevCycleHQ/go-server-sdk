package devcycle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

const CONFIG_RETRIES = 1

type ConfigReceiver interface {
	StoreConfig([]byte) error
}

type EnvironmentConfigManager struct {
	sdkKey              string
	configETag          string
	localBucketing      ConfigReceiver
	bucketingObjectPool ConfigReceiver
	firstLoad           bool
	context             context.Context
	stopPolling         context.CancelFunc
	httpClient          *http.Client
	cfg                 *HTTPConfiguration
	hasConfig           atomic.Bool
	ticker              *time.Ticker
}

func NewEnvironmentConfigManager(
	sdkKey string,
	localBucketing ConfigReceiver,
	bucketingObjectPool ConfigReceiver,
	options *DVCOptions,
	cfg *HTTPConfiguration,
) (e *EnvironmentConfigManager) {
	configManager := &EnvironmentConfigManager{
		sdkKey:              sdkKey,
		localBucketing:      localBucketing,
		bucketingObjectPool: bucketingObjectPool,
		cfg:                 cfg,
		httpClient:          &http.Client{Timeout: options.RequestTimeout},
		hasConfig:           atomic.Bool{},
		firstLoad:           true,
	}

	configManager.context, configManager.stopPolling = context.WithCancel(context.Background())

	return configManager
}

func (e *EnvironmentConfigManager) StartPolling(
	interval time.Duration,
) {
	e.ticker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-e.context.Done():
				warnf("Stopping config polling.")
				e.ticker.Stop()
				return
			case <-e.ticker.C:
				err := e.fetchConfig(CONFIG_RETRIES)
				if err != nil {
					warnf("Error fetching config: %s\n", err)
				}
			}
		}
	}()
}

func (e *EnvironmentConfigManager) initialFetch() error {
	return e.fetchConfig(CONFIG_RETRIES)
}

func (e *EnvironmentConfigManager) fetchConfig(numRetriesRemaining int) error {
	req, err := http.NewRequest("GET", e.getConfigURL(), nil)
	if err != nil {
		return err
	}

	if e.configETag != "" {
		req.Header.Set("If-None-Match", e.configETag)
	}
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch statusCode := resp.StatusCode; {
	case statusCode == http.StatusOK:
		if err = e.setConfigFromResponse(resp); err != nil {
			return err
		}
		return nil
	case statusCode == http.StatusNotModified:
		return nil
	case statusCode == http.StatusForbidden:
		e.stopPolling()
		return errorf("invalid SDK key. Aborting config polling")
	case statusCode >= 500:
		// Retryable Errors. Continue polling.
		if numRetriesRemaining > 0 {
			warnf("Retrying config fetch %d more times. Status: %s", numRetriesRemaining, resp.Status)
			return e.fetchConfig(numRetriesRemaining - 1)
		}
		warnf("Config fetch failed. Status:" + resp.Status)
	default:
		// TODO: Do we want to retry here as well?
		err = errorf("Unexpected response code: %d\n"+
			"Body: %s\n"+
			"URL: %s\n"+
			"Headers: %s\n"+
			"Could not download configuration. Using cached version if available %s\n",
			resp.StatusCode, resp.Body, e.getConfigURL(), resp.Header, resp.Header.Get("ETag"))
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
		return errorf("invalid JSON data received for config")
	}

	err = e.setConfig(config)

	if err != nil {
		return err
	}

	e.configETag = response.Header.Get("Etag")
	infof("Config set. ETag: %s\n", e.configETag)
	if e.firstLoad {
		e.firstLoad = false
		infof("DevCycle SDK Initialized.")
	}
	return nil
}

func (e *EnvironmentConfigManager) setConfig(config []byte) (err error) {
	err = e.localBucketing.StoreConfig(config)
	if err != nil {
		return
	}

	err = e.bucketingObjectPool.StoreConfig(config)
	if err != nil {
		return
	}

	e.hasConfig.Store(true)
	return
}

func (e *EnvironmentConfigManager) getConfigURL() string {
	configBasePath := e.cfg.ConfigCDNBasePath

	return fmt.Sprintf("%s/config/v1/server/%s.json", configBasePath, e.sdkKey)
}

func (e *EnvironmentConfigManager) HasConfig() bool {
	return e.hasConfig.Load()
}

func (e *EnvironmentConfigManager) Close() {
	e.stopPolling()
}
