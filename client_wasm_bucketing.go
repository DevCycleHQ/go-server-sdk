//go:build !native_bucketing

package devcycle

const NATIVE_SDK = false

func (c *Client) setLBClient(sdkKey string, options *Options) error {
	localBucketing, err := NewWASMLocalBucketing(sdkKey, c.platformData, options)
	if err != nil {
		return err
	}

	c.localBucketing = localBucketing

	eventQueue := &EventQueue{}
	err = eventQueue.initialize(options, localBucketing.localBucketingClient, localBucketing.bucketingObjectPool, c.cfg)

	if err != nil {
		return err
	}

	c.eventQueue = eventQueue

	return nil
}
