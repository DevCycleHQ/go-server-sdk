//go:build native_bucketing

package devcycle

import (
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"sync"

	"github.com/devcyclehq/go-server-sdk/v2/native_bucketing"
)

const NATIVE_SDK = true

func (c *DVCClient) setLBClient(sdkKey string, platformData *PlatformData, options *DVCOptions) error {
	localBucketing := NewNativeLocalBucketing(sdkKey, options)
	localBucketing.SetPlatformData(platformData)
	c.localBucketing = localBucketing

	// Event queue stub that does nothing
	c.eventQueue = &NativeEventQueue{}

	return nil
}

type NativeLocalBucketing struct {
	sdkKey      string
	options     *DVCOptions
	configMutex sync.RWMutex
}

func NewNativeLocalBucketing(sdkKey string, options *DVCOptions) *NativeLocalBucketing {
	return &NativeLocalBucketing{
		sdkKey:  sdkKey,
		options: options,
	}
}

func (n *NativeLocalBucketing) StoreConfig(configJSON []byte, eTag string) error {
	err := native_bucketing.SetConfig(configJSON, n.sdkKey, eTag)
	if err != nil {
		return fmt.Errorf("Error parsing config: %w", err)
	}
	return nil
}

func (n *NativeLocalBucketing) GenerateBucketedConfigForUser(user DVCUser) (ret *BucketedUserConfig, err error) {
	platformData, err := native_bucketing.GetPlatformData(n.sdkKey)
	if err != nil {
		return nil, err
	}
	populatedUser := user.GetPopulatedUser(platformData)
	clientCustomData := native_bucketing.GetClientCustomData(n.sdkKey)
	return native_bucketing.GenerateBucketedConfig(n.sdkKey, populatedUser, clientCustomData)
}

func (n *NativeLocalBucketing) SetClientCustomData(customData map[string]interface{}) error {
	native_bucketing.SetClientCustomData(n.sdkKey, customData)
	return nil
}

func (n *NativeLocalBucketing) Variable(user DVCUser, variableKey string, variableType string) (Variable, error) {
	defaultVar := Variable{
		BaseVariable: api.BaseVariable{
			Key:   variableKey,
			Type_: variableType,
			Value: nil,
		},
		DefaultValue: nil,
		IsDefaulted:  true,
	}
	clientCustomData := native_bucketing.GetClientCustomData(n.sdkKey)
	platformData, err := native_bucketing.GetPlatformData(n.sdkKey)
	if err != nil {
		return defaultVar, err
	}

	populatedUser := user.GetPopulatedUser(platformData)
	variable, err := native_bucketing.VariableForUser(n.sdkKey, populatedUser, variableKey, variableType, false, clientCustomData)
	if err != nil {
		return defaultVar, err
	}

	return Variable{
		BaseVariable: variable.BaseVariable,
		IsDefaulted:  false,
	}, nil
}

func (N *NativeLocalBucketing) SetPlatformData(platformData *PlatformData) {
	native_bucketing.SetPlatformData(N.sdkKey, *platformData)
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
