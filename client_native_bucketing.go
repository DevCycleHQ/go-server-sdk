//go:build !devcycle_wasm_bucketing

package devcycle

import (
	"fmt"
	"sync"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/util"

	"github.com/devcyclehq/go-server-sdk/v2/api"

	"github.com/devcyclehq/go-server-sdk/v2/bucketing"
)

const NATIVE_SDK = true

// This value will always be set to zero as the user.CreatedDate is not actually used in native bucketing
var DEFAULT_USER_TIME = time.Time{}

func (c *Client) setLBClient(sdkKey string, options *Options) error {
	localBucketing, err := NewNativeLocalBucketing(sdkKey, c.platformData, options)
	if err != nil {
		return err
	}
	c.localBucketing = localBucketing

	return nil
}

type NativeLocalBucketing struct {
	sdkKey       string
	options      *Options
	configMutex  sync.RWMutex
	platformData *api.PlatformData
	eventQueue   *bucketing.EventQueue
}

func NewNativeLocalBucketing(sdkKey string, platformData *api.PlatformData, options *Options) (*NativeLocalBucketing, error) {
	eq, err := bucketing.NewEventQueue(sdkKey, options.eventQueueOptions(), platformData)
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
	err := bucketing.SetConfig(configJSON, n.sdkKey, eTag, n.eventQueue)
	if err != nil {
		return fmt.Errorf("Error parsing config: %w", err)
	}
	return nil
}

func (n *NativeLocalBucketing) GenerateBucketedConfigForUser(user User) (ret *BucketedUserConfig, err error) {
	populatedUser := user.GetPopulatedUserWithTime(n.platformData, DEFAULT_USER_TIME)
	clientCustomData := bucketing.GetClientCustomData(n.sdkKey)
	return bucketing.GenerateBucketedConfig(n.sdkKey, populatedUser, clientCustomData)
}

func (n *NativeLocalBucketing) SetClientCustomData(customData map[string]interface{}) error {
	bucketing.SetClientCustomData(n.sdkKey, customData)
	return nil
}

func (n *NativeLocalBucketing) Variable(user User, variableKey string, variableType string) (Variable, error) {

	defaultVar := Variable{
		BaseVariable: api.BaseVariable{
			Key:   variableKey,
			Type_: variableType,
			Value: nil,
		},
		DefaultValue: nil,
		IsDefaulted:  true,
	}
	clientCustomData := bucketing.GetClientCustomData(n.sdkKey)
	populatedUser := user.GetPopulatedUserWithTime(n.platformData, DEFAULT_USER_TIME)

	resultVariableType, resultValue, err := bucketing.VariableForUser(n.sdkKey, populatedUser, variableKey, variableType, n.eventQueue, clientCustomData)
	if err != nil {
		return defaultVar, nil
	}

	return Variable{
		BaseVariable: api.BaseVariable{
			Key:   variableKey,
			Type_: resultVariableType,
			Value: resultValue,
		},
		IsDefaulted: false,
	}, nil
}

func (n *NativeLocalBucketing) Close() {
	err := n.eventQueue.Close()
	if err != nil {
		util.Errorf("Error closing event queue: %v", err)
	}
}

func (n *NativeLocalBucketing) QueueEvent(user User, event Event) error {
	return n.eventQueue.QueueEvent(user, event)
}

func (n *NativeLocalBucketing) QueueAggregateEvent(config BucketedUserConfig, event Event) error {
	return n.eventQueue.QueueAggregateEvent(config, event)
}

func (n *NativeLocalBucketing) UserQueueLength() (int, error) {
	return n.eventQueue.UserQueueLength(), nil
}

func (n *NativeLocalBucketing) FlushEventQueue(callback EventFlushCallback) error {
	payloads, err := n.eventQueue.FlushEventQueue()
	if err != nil {
		return fmt.Errorf("Error flushing event queue: %w", err)
	}

	result, err := callback(payloads)
	if err != nil {
		return err
	}

	n.eventQueue.HandleFlushResults(result.SuccessPayloads, result.FailurePayloads, result.FailureWithRetryPayloads)

	return nil
}

func (n *NativeLocalBucketing) Metrics() (int32, int32, int32) {
	return n.eventQueue.Metrics()
}
