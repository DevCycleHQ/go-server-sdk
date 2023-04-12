//go:build native_bucketing

package devcycle

import (
	"fmt"
	"sync"

	"github.com/devcyclehq/go-server-sdk/v2/native_bucketing"
)

const NATIVE_SDK = true

func (c *DVCClient) setLBClient(sdkKey string, options *DVCOptions) error {
	localBucketing := NewNativeLocalBucketing(sdkKey, options)
	c.localBucketing = localBucketing

	// Event queue stub that does nothing
	c.eventQueue = &NativeEventQueue{}

	return nil
}

type NativeLocalBucketing struct {
	sdkKey      string
	options     *DVCOptions
	configMutex sync.RWMutex
	config      *native_bucketing.ConfigBody
}

func NewNativeLocalBucketing(sdkKey string, options *DVCOptions) *NativeLocalBucketing {
	return &NativeLocalBucketing{
		sdkKey:  sdkKey,
		options: options,
	}
}

func (n *NativeLocalBucketing) StoreConfig(configJSON []byte) error {
	// TODO: How do we get the ETag here?
	eTag := ""
	config, err := native_bucketing.NewConfig(configJSON, eTag)
	if err != nil {
		return fmt.Errorf("Error parsing config: %w", err)
	}

	n.configMutex.Lock()
	defer n.configMutex.Unlock()
	n.config = &config

	return nil
}

func (n *NativeLocalBucketing) GetConfig() native_bucketing.ConfigBody {
	n.configMutex.RLock()
	defer n.configMutex.RUnlock()
	return *n.config
}

func (n *NativeLocalBucketing) GenerateBucketedConfigForUser(user DVCUser) (ret *BucketedUserConfig, err error) {
	config := n.GetConfig()
	populatedUser := user.GetPopulatedUser()
	return native_bucketing.GenerateBucketedConfig(config, populatedUser, nil)
}

func (n *NativeLocalBucketing) SetClientCustomData(customData map[string]interface{}) error {
	// TODO: implement
	return fmt.Errorf("not implemented")
}

func (n *NativeLocalBucketing) Variable(user DVCUser, variableKey string, variableType string) (Variable, error) {
	defaultVar := Variable{
		IsDefaulted: true,
	}
	variable, err := native_bucketing.VariableForUser(n.GetConfig(), n.sdkKey, user.GetPopulatedUser(), variableKey, variableType, false, nil)
	if err != nil {
		return defaultVar, err
	}

	return Variable{
		BaseVariable: variable.BaseVariable,
		IsDefaulted:  false,
	}, nil
}

func (n *NativeLocalBucketing) Close() {
	// TODO: implement
}

type NativeEventQueue struct{}

func (queue *NativeEventQueue) QueueEvent(user DVCUser, event DVCEvent) error {
	// TODO: implement
	return nil
}

func (queue *NativeEventQueue) QueueAggregateEvent(config BucketedUserConfig, event DVCEvent) error {
	// TODO: implement
	return nil
}

func (queue *NativeEventQueue) FlushEvents() (err error) {
	// TODO: implement
	return nil
}

func (queue *NativeEventQueue) Metrics() (int32, int32) {
	// TODO: implement
	return 0, 0
}

func (queue *NativeEventQueue) Close() (err error) {
	// TODO: implement
	return nil
}
