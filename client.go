package devcycle

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

var (
	jsonCheck = regexp.MustCompile("(?i:[application|text]/json)")
	xmlCheck  = regexp.MustCompile("(?i:[application|text]/xml)")
)

// DVCClient
// In most cases there should be only one, shared, DVCClient.
type DVCClient struct {
	cfg    *HTTPConfiguration
	common service // Reuse a single struct instead of allocating one for each service on the heap.

	// API Services
	DevCycleApi     *DVCClientService
	DevCycleOptions *DVCOptions
	environmentKey  string
	auth            context.Context
	localBucketing  *DevCycleLocalBucketing
	configManager   *EnvironmentConfigManager
	eventQueue      *EventQueue
	isInitialized   bool
}

type SDKEvent struct {
	Success             bool   `json:"success"`
	Message             string `json:"message"`
	Error               error  `json:"error"`
	FirstInitialization bool   `json:"firstInitialization"`
}

type service struct {
	client *DVCClient
}

func initializeLocalBucketing(environmentKey string, options *DVCOptions) (ret *DevCycleLocalBucketing, err error) {
	cfg := NewConfiguration(options)

	options.CheckDefaults()
	ret = &DevCycleLocalBucketing{}
	err = ret.Initialize(environmentKey, options, cfg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return
}

func setLBClient(environmentKey string, options *DVCOptions, c *DVCClient) (*DVCClient, error) {
	localBucketing, err := initializeLocalBucketing(environmentKey, options)

	if err != nil {
		if options.OnInitializedChannel != nil {
			options.OnInitializedChannel <- true
		}
		return nil, err
	}
	c.localBucketing = localBucketing
	c.configManager = c.localBucketing.configManager
	c.eventQueue = c.localBucketing.eventQueue
	c.isInitialized = true
	if options.OnInitializedChannel != nil {
		options.OnInitializedChannel <- true
		close(options.OnInitializedChannel)
	}
	return c, nil
}

// NewDVCClient creates a new API client.
// optionally pass a custom http.Client to allow for advanced features such as caching.
func NewDVCClient(environmentKey string, options *DVCOptions) (*DVCClient, error) {
	if environmentKey == "" {
		return nil, fmt.Errorf("Missing environment key! Call NewDVCClient with a valid environment key.")
	}
	if !environmentKeyIsValid(environmentKey) {
		return nil, fmt.Errorf("Invalid environment key. Call NewDVCClient with a valid environment key.")
	}
	cfg := NewConfiguration(options)

	options.CheckDefaults()

	c := &DVCClient{environmentKey: environmentKey}
	c.cfg = cfg
	c.common.client = c
	// API Services
	c.DevCycleApi = (*DVCClientService)(&c.common)

	c.DevCycleOptions = options

	if !c.DevCycleOptions.EnableCloudBucketing {
		if c.DevCycleOptions.OnInitializedChannel != nil {
			go setLBClient(environmentKey, options, c)
		} else {
			return setLBClient(environmentKey, options, c)
		}
	}
	return c, nil
}

// callAPI do the request.
func (c *DVCClient) callAPI(request *http.Request) (*http.Response, error) {
	return c.cfg.HTTPClient.Do(request)
}

// Change base path to allow switching to mocks
func (c *DVCClient) ChangeBasePath(path string) {
	c.cfg.BasePath = path
}

func (c *DVCClient) SetOptions(dvcOptions DVCOptions) {
	c.DevCycleOptions = &dvcOptions
}

// prepareRequest build the request
func (c *DVCClient) prepareRequest(
	path string,
	method string,
	postBody interface{},
	headerParams map[string]string,
	queryParams url.Values,
) (localVarRequest *http.Request, err error) {

	var body *bytes.Buffer

	// Detect postBody type and post.
	if postBody != nil {
		contentType := headerParams["Content-Type"]
		if contentType == "" {
			contentType = detectContentType(postBody)
			headerParams["Content-Type"] = contentType
		}

		body, err = setBody(postBody, contentType)
		if err != nil {
			return nil, err
		}
	}

	// Setup path and query parameters
	url, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	// Adding Query Param
	query := url.Query()
	for k, v := range queryParams {
		for _, iv := range v {
			query.Add(k, iv)
		}
	}

	if c.DevCycleOptions.EnableEdgeDB {
		query.Add("enableEdgeDB", "true")
	}

	// Encode the parameters.
	url.RawQuery = query.Encode()

	// Generate a new request
	if body != nil {
		localVarRequest, err = http.NewRequest(method, url.String(), body)
	} else {
		localVarRequest, err = http.NewRequest(method, url.String(), nil)
	}
	if err != nil {
		return nil, err
	}

	// add header parameters, if any
	if len(headerParams) > 0 {
		headers := http.Header{}
		for h, v := range headerParams {
			headers.Set(h, v)
		}
		localVarRequest.Header = headers
	}

	// Override request host, if applicable
	if c.cfg.Host != "" {
		localVarRequest.Host = c.cfg.Host
	}

	// Add the user agent to the request.
	localVarRequest.Header.Add("User-Agent", c.cfg.UserAgent)

	for header, value := range c.cfg.DefaultHeader {
		localVarRequest.Header.Add(header, value)
	}

	return localVarRequest, nil
}

func (c *DVCClient) decode(v interface{}, b []byte, contentType string) (err error) {
	if strings.Contains(contentType, "application/xml") {
		if err = xml.Unmarshal(b, v); err != nil {
			return err
		}
		return nil
	} else if strings.Contains(contentType, "application/json") {
		if err = json.Unmarshal(b, v); err != nil {
			return err
		}
		return nil
	}
	return errors.New("undefined response type")
}

// Set request body from an interface{}
func setBody(body interface{}, contentType string) (bodyBuf *bytes.Buffer, err error) {
	if bodyBuf == nil {
		bodyBuf = &bytes.Buffer{}
	}

	if reader, ok := body.(io.Reader); ok {
		_, err = bodyBuf.ReadFrom(reader)
	} else if b, ok := body.([]byte); ok {
		_, err = bodyBuf.Write(b)
	} else if s, ok := body.(string); ok {
		_, err = bodyBuf.WriteString(s)
	} else if s, ok := body.(*string); ok {
		_, err = bodyBuf.WriteString(*s)
	} else if jsonCheck.MatchString(contentType) {
		err = json.NewEncoder(bodyBuf).Encode(body)
	} else if xmlCheck.MatchString(contentType) {
		xml.NewEncoder(bodyBuf).Encode(body)
	}

	if err != nil {
		return nil, err
	}

	if bodyBuf.Len() == 0 {
		err = fmt.Errorf("Invalid body type %s\n", contentType)
		return nil, err
	}
	return bodyBuf, nil
}

// detectContentType method is used to figure out `Request.Body` content type for request header
func detectContentType(body interface{}) string {
	contentType := "text/plain; charset=utf-8"
	kind := reflect.TypeOf(body).Kind()

	switch kind {
	case reflect.Struct, reflect.Map, reflect.Ptr:
		contentType = "application/json; charset=utf-8"
	case reflect.String:
		contentType = "text/plain; charset=utf-8"
	default:
		if b, ok := body.([]byte); ok {
			contentType = http.DetectContentType(b)
		} else if kind == reflect.Slice {
			contentType = "application/json; charset=utf-8"
		}
	}

	return contentType
}

// GenericSwaggerError Provides access to the body, error and model on returned errors.
type GenericSwaggerError struct {
	body  []byte
	error string
	model interface{}
}

// Error returns non-empty string if there was an error.
func (e GenericSwaggerError) Error() string {
	return e.error
}

// Body returns the raw bytes of the response
func (e GenericSwaggerError) Body() []byte {
	return e.body
}

// Model returns the unpacked model of the error
func (e GenericSwaggerError) Model() interface{} {
	return e.model
}

func environmentKeyIsValid(key string) bool {
	return strings.HasPrefix(key, "server") || strings.HasPrefix(key, "dvc_server")
}
