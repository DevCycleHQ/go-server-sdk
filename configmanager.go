package devcycle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DevCycleHQ/tunny"
)

type EnvironmentConfigManager struct {
	sdkKey              string
	configETag          string
	localBucketing      *DevCycleLocalBucketing
	bucketingWorkers    []*LocalBucketingWorker
	bucketingWorkerPool *tunny.Pool
	firstLoad           bool
	context             context.Context
	cancel              context.CancelFunc
	httpClient          *http.Client
	cfg                 *HTTPConfiguration
	hasConfig           bool
	pollingStop         chan bool
	ticker              *time.Ticker
}

func (e *EnvironmentConfigManager) Initialize(
	sdkKey string,
	localBucketing *DevCycleLocalBucketing,
	bucketingWorkers []*LocalBucketingWorker,
	bucketingWorkerPool *tunny.Pool,
	cfg *HTTPConfiguration,
) (err error) {
	e.localBucketing = localBucketing
	e.bucketingWorkers = bucketingWorkers
	e.bucketingWorkerPool = bucketingWorkerPool
	e.sdkKey = sdkKey
	e.cfg = cfg
	e.httpClient = &http.Client{Timeout: localBucketing.options.RequestTimeout}
	e.context, e.cancel = context.WithCancel(context.Background())
	e.pollingStop = make(chan bool, 2)

	e.firstLoad = true

	e.ticker = time.NewTicker(e.localBucketing.options.ConfigPollingIntervalMS)

	go func() {
		for {
			select {
			case <-e.pollingStop:
				warnf("Stopping config polling.")
				e.ticker.Stop()
				return
			case <-e.ticker.C:
				err = e.fetchConfig(false)
				if err != nil {
					warnf("Error fetching config: %s\n", err)
				}
			}
		}
	}()
	return
}

func (e *EnvironmentConfigManager) initialFetch() (err error) {
	err = e.fetchConfig(false)
	return
}

func (e *EnvironmentConfigManager) fetchConfig(retrying bool) error {
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
		break
	case statusCode == http.StatusNotModified:
		break
	case statusCode == http.StatusForbidden:
		e.pollingStop <- true
		return errorf("invalid SDK key. Aborting config polling")
	case statusCode >= 500:
		// Retryable Errors. Continue polling.
		if !retrying {
			warnf("Retrying config fetch. Status:" + resp.Status)
			return e.fetchConfig(true)
		}
		warnf("Config fetch failed. Status:" + resp.Status)
		break
	default:
		err = errorf("Unexpected response code: %d\n"+
			"Body: %s\n"+
			"URL: %s\n"+
			"Headers: %s\n"+
			"Could not download configuration. Using cached version if available %s\n",
			resp.StatusCode, resp.Body, e.getConfigURL(), resp.Header, resp.Header.Get("ETag"))
		e.context.Done()
		e.cancel()
		break
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

	if e.bucketingWorkerPool != nil {
		errs := e.bucketingWorkerPool.ProcessAll(&WorkerPoolPayload{
			Type_:      "storeConfig",
			ConfigData: &config,
		})

		for _, err := range errs {
			var response = err.(WorkerPoolResponse)
			if response.Err != nil {
				return response.Err
			}
		}
	}

	e.hasConfig = true
	return
}

func (e *EnvironmentConfigManager) getConfigURL() string {
	configBasePath := e.cfg.ConfigCDNBasePath

	return fmt.Sprintf("%s/config/v1/server/%s.json", configBasePath, e.sdkKey)
}

func (e *EnvironmentConfigManager) HasConfig() bool {
	return e.hasConfig
}

func (e *EnvironmentConfigManager) Close() {
	e.pollingStop <- true
}
