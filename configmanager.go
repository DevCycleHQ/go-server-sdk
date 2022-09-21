package devcycle

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

var pollingStop = make(chan bool, 1)

type EnvironmentConfigManager struct {
	environmentKey string
	configETag     string
	localBucketing *DevCycleLocalBucketing
	firstLoad      bool
	context        context.Context
	cancel         context.CancelFunc
	httpClient     *http.Client
	httpConfig     *HTTPConfiguration
	options        *DVCOptions
}

func (e *EnvironmentConfigManager) Initialize(environmentKey string, options *DVCOptions) (err error) {
	e.options = options
	e.environmentKey = environmentKey
	e.httpClient = &http.Client{Timeout: options.RequestTimeout}
	e.context, e.cancel = context.WithCancel(context.Background())

	ticker := time.NewTicker(options.PollingInterval)
	e.firstLoad = true

	err = e.fetchConfig()
	if err != nil {
		return err
	}

	go func(ctx context.Context) {
		for {
			select {
			case <-pollingStop:
			case <-ctx.Done():
				ticker.Stop()
				log.Println("Stopping config polling.")
				return
			case <-ticker.C:
				err = e.fetchConfig()
				if err != nil {
					log.Printf("Error fetching config: %s\n", err)
				}
			}
		}
	}(e.context)
	return nil
}

func (e *EnvironmentConfigManager) fetchConfig() error {
	req, err := http.NewRequest("GET", e.getConfigURL(), nil)
	if e.configETag != "" {
		req.Header.Set("If-None-Match", e.configETag)
	}
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		err = e.setConfig(resp)
		if err != nil {
			return err
		}
		break
	case http.StatusNotModified:
		log.Printf("Config not modified. Using cached config. %s\n", e.configETag)
		break
	case http.StatusForbidden:
		pollingStop <- true
		return fmt.Errorf("403 Forbidden - SDK key is likely incorrect. Aborting polling")

	case http.StatusInternalServerError:
	case http.StatusBadGateway:
	case http.StatusServiceUnavailable:
		// Retryable Errors. Continue polling.
		log.Println("Retrying config fetch. Status:" + resp.Status)
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

	config := string(raw)
	err = e.localBucketing.StoreConfig(e.environmentKey, config)
	if err != nil {
		return err
	}
	e.configETag = response.Header.Get("Etag")
	log.Printf("Config set. ETag: %s\n", e.configETag)
	if e.firstLoad {
		e.firstLoad = false
		log.Println("DevCycle SDK Initialized.")
	}
	return nil
}

func (e *EnvironmentConfigManager) getConfigURL() string {
	if e.options.ConfigCDNOverride != "" {
		return e.options.ConfigCDNOverride
	}
	return fmt.Sprintf("https://config-cdn.devcycle.com/config/v1/server/%s.json", e.environmentKey)
}
