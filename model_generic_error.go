package devcycle

// GenericError Provides access to the body, error and model on returned errors.
type GenericError struct {
	body  []byte
	error string
	model interface{}
}

// Error returns non-empty string if there was an error.
func (e GenericError) Error() string {
	return e.error
}

// Body returns the raw bytes of the response
func (e GenericError) Body() []byte {
	return e.body
}

// Model returns the unpacked model of the error
func (e GenericError) Model() interface{} {
	return e.model
}
