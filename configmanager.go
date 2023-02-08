package devcycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type EnvironmentConfigManager struct {
	sdkKey         string
	configETag     string
	localBucketing *DevCycleLocalBucketing
	firstLoad      bool
	context        context.Context
	cancel         context.CancelFunc
	httpClient     *http.Client
	hasConfig      bool
	pollingStop    chan bool
}

func (e *EnvironmentConfigManager) Initialize(sdkKey string, localBucketing *DevCycleLocalBucketing) (err error) {
	e.localBucketing = localBucketing
	e.sdkKey = sdkKey
	e.httpClient = &http.Client{Timeout: localBucketing.options.RequestTimeout}
	e.context, e.cancel = context.WithCancel(context.Background())
	e.pollingStop = make(chan bool, 2)

	ticker := time.NewTicker(localBucketing.options.ConfigPollingIntervalMS)
	e.firstLoad = true

	err = e.fetchConfig(false)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-e.pollingStop:
				log.Printf("Stopping config polling.")
				ticker.Stop()
				return
			case <-ticker.C:
				err = e.fetchConfig(false)
				if err != nil {
					log.Printf("Error fetching config: %s\n", err)
				}
			}
		}
	}()

	return nil
}

func (e *EnvironmentConfigManager) fetchConfig(retrying bool) error {
	req, err := http.NewRequest("GET", e.getConfigURL(), nil)
	if e.configETag != "" {
		req.Header.Set("If-None-Match", e.configETag)
	}
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	switch statusCode := resp.StatusCode; {
	case statusCode == http.StatusOK:
		err = e.setConfig(resp)
		if err != nil {
			return err
		}
		break
	case statusCode == http.StatusNotModified:
		log.Printf("Config not modified. Using cached config. %s\n", e.configETag)
		break
	case statusCode == http.StatusForbidden:
		e.pollingStop <- true
		return fmt.Errorf("invalid SDK key. Aborting config polling")
	case statusCode >= 500:
		// Retryable Errors. Continue polling.
		if !retrying {
			log.Println("Retrying config fetch. Status:" + resp.Status)
			return e.fetchConfig(true)
		}
		log.Println("Config fetch failed. Status:" + resp.Status)
		break
	default:
		log.Printf("Unexpected response code: %d\n", resp.StatusCode)
		log.Printf("Body: %s\n", resp.Body)
		log.Printf("URL: %s\n", e.getConfigURL())
		log.Printf("Headers: %s\n", resp.Header)
		log.Printf("Could not download configuration. Using cached version if available %s\n", resp.Header.Get("ETag"))
		e.context.Done()
		e.cancel()
		break
	}
	return nil
}

func (e *EnvironmentConfigManager) setConfig(response *http.Response) error {
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	// Check
	valid := json.Valid(raw)
	if !valid {
		return errors.New("invalid JSON data received for config")
	}

	config := string(raw)
	err = e.localBucketing.StoreConfig(e.sdkKey, config)
	if err != nil {
		return err
	}
	e.hasConfig = true
	e.configETag = response.Header.Get("Etag")
	log.Printf("Config set. ETag: %s\n", e.configETag)
	if e.firstLoad {
		e.firstLoad = false
		log.Println("DevCycle SDK Initialized.")
	}
	return nil
}

func (e *EnvironmentConfigManager) getConfigURL() string {
	configBasePath := e.localBucketing.cfg.ConfigCDNBasePath

	return fmt.Sprintf("%s/config/v1/server/%s.json", configBasePath, e.sdkKey)
}

func (e *EnvironmentConfigManager) HasConfig() bool {
	return e.hasConfig
}

func (e *EnvironmentConfigManager) Close() {
	e.pollingStop <- true
}
