package devcycle

import (
	"context"
	"fmt"
	"google.golang.org/appengine/log"
	"io"
	"net/http"
	"strconv"
	"time"
)

var configMap map[string]Configuration
var pollingStop = make(chan bool)

type SDKEvent struct {
	Success             bool   `json:"success"`
	Message             string `json:"message"`
	Error               error  `json:"error"`
	FirstInitialization bool   `json:"firstInitialization"`
}

type EnvironmentConfigManager struct {
	EnvironmentKey string
	configETag     string
	LocalBucketing *DevCycleLocalBucketing
	firstLoad      bool
	SDKEvents      chan SDKEvent
}

func (e *EnvironmentConfigManager) Initialize() {
	//if e.LocalBucketing.po == 0 {
	//	e.PollingInterval = time.Second * 30
	//}
	//if e.RequestTimeout == 0 {
	//	e.RequestTimeout = time.Second * 10
	//}

	ticker := time.NewTicker(10 * time.Second)
	e.firstLoad = true

	go func() {
		for {
			select {
			case <-pollingStop:
				ticker.Stop()
				log.Criticalf(context.Background(), "Stopping config polling.")
				return
			case <-ticker.C:
				e.fetchConfig()
			}
		}
	}()
}

func (e *EnvironmentConfigManager) fetchConfig() {
	resp, err := http.Get(e.getConfigURL())
	if err != nil {
		e.SDKEvents <- SDKEvent{Success: false, Message: "Could not make HTTP Request to CDN.", Error: err}
	}
	switch resp.StatusCode {
	case http.StatusOK:
		err := e.setConfig(resp)
		if err != nil {
			e.SDKEvents <- SDKEvent{Success: false, Message: "Failed to set config.", Error: err}
			return
		}
		break
	case http.StatusNotModified:
		log.Infof(context.Background(), "Config not modified. Using cached config. %s", e.configETag)
		break
	case http.StatusForbidden:
		log.Errorf(context.Background(), "403 Forbidden - SDK key is likely incorrect. Aborting polling.")
		pollingStop <- true
		return
	case http.StatusInternalServerError:
	case http.StatusBadGateway:
	case http.StatusServiceUnavailable:
		// Retryable Errors. Continue polling.
		log.Warningf(context.Background(), "Retrying config fetch. Status: %s", resp.Status)
		break
	default:
		log.Errorf(context.Background(), "Unexpected response code: %d", resp.StatusCode)
		log.Errorf(context.Background(), "Body: %s", resp.Body)
		log.Errorf(context.Background(), "URL: %s", e.getConfigURL())
		log.Errorf(context.Background(), "Headers: %s", resp.Header)
		log.Errorf(context.Background(), "Could not download configuration. Using cached version if available %s", resp.Header.Get("ETag"))
		e.SDKEvents <- SDKEvent{Success: false,
			Message: "Unexpected response code - Aborting Polling. Code: " + strconv.Itoa(resp.StatusCode), Error: nil}
		pollingStop <- true
		break
	}
}

func (e *EnvironmentConfigManager) setConfig(response *http.Response) error {
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	err = e.LocalBucketing.StoreConfig(e.EnvironmentKey, string(raw))
	if err != nil {
		return err
	}
	e.configETag = response.Header.Get("ETag")
	log.Infof(context.Background(), "Config set. ETag: %s", e.configETag)
	e.SDKEvents <- SDKEvent{Success: true, Message: "Config set. ETag: " + e.configETag, Error: nil}
	if e.firstLoad {
		e.firstLoad = false
		log.Infof(context.Background(), "DevCycle SDK Initialized.")
		e.SDKEvents <- SDKEvent{Success: true, Message: "DevCycle SDK Initialized.", Error: nil, FirstInitialization: true}
	}
	return nil
}

func (e *EnvironmentConfigManager) getConfigURL() string {
	return fmt.Sprintf("https://config-cdn.devcycle.com/config/v1/server/%s.json", e.EnvironmentKey)
}
