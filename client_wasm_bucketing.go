package devcycle

func (c *DVCClient) setLBClient(sdkKey string, options *DVCOptions) error {
	localBucketing, err := NewWASMLocalBucketing(sdkKey, options)
	if err != nil {
		return err
	}
	c.localBucketing = localBucketing

	c.eventQueue = &EventQueue{}
	err = c.eventQueue.initialize(options, localBucketing.localBucketingClient, localBucketing.bucketingObjectPool, c.cfg)

	if err != nil {
		return err
	}

	c.configManager = NewEnvironmentConfigManager(sdkKey, localBucketing, options, c.cfg)
	c.configManager.StartPolling(options.ConfigPollingIntervalMS)

	if err != nil {
		return err
	}

	return err
}
