package devcycle

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

var pollingStop = make(chan bool)

type EnvironmentConfigManager struct {
	environmentKey string
	configETag     string
	localBucketing *DevCycleLocalBucketing
	firstLoad      bool
	SDKEvents      chan SDKEvent
	context        context.Context
	cancel         context.CancelFunc
}

func (e *EnvironmentConfigManager) Initialize(environmentKey string, options *DVCOptions) (err error) {
	e.environmentKey = environmentKey
	e.context, e.cancel = context.WithCancel(context.Background())
	e.SDKEvents = make(chan SDKEvent, 100)

	if options.PollingInterval == 0 {
		options.PollingInterval = time.Second * 30
	}
	if options.RequestTimeout == 0 {
		options.RequestTimeout = time.Second * 10
	}

	ticker := time.NewTicker(options.PollingInterval)
	e.firstLoad = true

	err = e.fetchConfig()
	if err != nil {
		fmt.Println(err)
		return err
	}

	go func(ctx context.Context) {
		for {
			select {
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
	resp, err := http.Get(e.getConfigURL())
	if err != nil {
		e.SDKEvents <- SDKEvent{Success: false, Message: "Could not make HTTP Request to CDN.", Error: err}
		return err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		err = e.setConfig(resp)
		if err != nil {
			e.SDKEvents <- SDKEvent{Success: false, Message: "Failed to set config.", Error: err}
			return err
		}
		break
	case http.StatusNotModified:
		log.Printf("Config not modified. Using cached config. %s\n", e.configETag)
		break
	case http.StatusForbidden:
		pollingStop <- true
		log.Println("403 Forbidden - SDK key is likely incorrect. Aborting polling.")
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
		e.SDKEvents <- SDKEvent{Success: false,
			Message: "Unexpected response code - Aborting Polling. Code: " + strconv.Itoa(resp.StatusCode), Error: nil}
		e.context.Done()
		e.cancel()
		break
	}
	fmt.Println("Config fetch complete.")
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
	//e.SDKEvents <- SDKEvent{Success: true, Message: "Config set. ETag: " + e.configETag, Error: nil}
	if e.firstLoad {
		e.firstLoad = false
		log.Println("DevCycle SDK Initialized.")
		select {
		case e.SDKEvents <- SDKEvent{Success: true, Message: "DevCycle SDK Initialized.", Error: nil, FirstInitialization: true}:
			break
		default:
			log.Println("No listener for SDK Events. Not sending events.")
		}
	}
	return nil
}

func (e *EnvironmentConfigManager) getConfigURL() string {
	return fmt.Sprintf("https://config-cdn.devcycle.com/config/v1/server/%s.json", e.environmentKey)
}
