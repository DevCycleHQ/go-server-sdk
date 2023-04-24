//go:build native_bucketing

package devcycle

import (
	"fmt"
	"sync"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"

	"github.com/devcyclehq/go-server-sdk/v2/native_bucketing"
)

const NATIVE_SDK = true

// This value will always be set to zero as the user.CreatedDate is not actually used in native bucketing
var DEFAULT_USER_TIME = time.Time{}

func (c *DVCClient) setLBClient(sdkKey string, options *DVCOptions) error {
	localBucketing, err := NewNativeLocalBucketing(sdkKey, c.platformData, options)
	if err != nil {
		return err
	}
	c.localBucketing = localBucketing
	c.eventQueue = localBucketing.eventQueue

	return nil
}

type NativeLocalBucketing struct {
	sdkKey       string
	options      *DVCOptions
	configMutex  sync.RWMutex
	platformData *api.PlatformData
	eventQueue   *native_bucketing.EventQueue
}

func NewNativeLocalBucketing(sdkKey string, platformData *api.PlatformData, options *DVCOptions) (*NativeLocalBucketing, error) {
	// Event queue stub that does nothing
	eq, err := native_bucketing.InitEventQueue(sdkKey, options.eventQueueOptions())
	if err != nil {
		return nil, err
	}
	return &NativeLocalBucketing{
		sdkKey:       sdkKey,
		options:      options,
		platformData: platformData,
		eventQueue:   eq,
	}, err
}

func (n *NativeLocalBucketing) StoreConfig(configJSON []byte, eTag string) error {
	err := native_bucketing.SetConfig(configJSON, n.sdkKey, eTag, n.eventQueue)
	if err != nil {
		return fmt.Errorf("Error parsing config: %w", err)
	}
	return nil
}

func (n *NativeLocalBucketing) GenerateBucketedConfigForUser(user DVCUser) (ret *BucketedUserConfig, err error) {
	populatedUser := user.GetPopulatedUserWithTime(n.platformData, DEFAULT_USER_TIME)
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
	populatedUser := user.GetPopulatedUserWithTime(n.platformData, DEFAULT_USER_TIME)
	variable, err := native_bucketing.VariableForUser(n.sdkKey, populatedUser, variableKey, variableType, false, clientCustomData)
	if err != nil {
		// TODO: Are there errors that can be returned here that should be surfaced to the client?
		return defaultVar, nil
	}

	return Variable{
		BaseVariable: variable.BaseVariable,
		IsDefaulted:  false,
	}, nil
}

func (n *NativeLocalBucketing) Close() {
	//TODO: Implement
}
