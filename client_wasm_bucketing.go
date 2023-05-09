//go:build !native_bucketing

package devcycle

const NATIVE_SDK = false

func (c *Client) setLBClient(sdkKey string, options *Options) error {
	localBucketing, err := NewWASMLocalBucketing(sdkKey, c.platformData, options)
	if err != nil {
		return err
	}

	c.localBucketing = localBucketing

	return nil
}
