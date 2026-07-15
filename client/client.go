package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"

	"github.com/courtyard-nft/go-shippo/errors"
	"github.com/courtyard-nft/go-shippo/models"
)

const (
	shippoAPIBaseURL = "https://api.goshippo.com/v1"
)

type Client struct {
	privateToken string
	apiVersion   string
	logger       *log.Logger
	baseURL      string
	httpClient   *http.Client
}

type listOutputCallback func(v json.RawMessage) error

// listOutputCallbackWithStop processes a single list item and returns
// stop=true to end pagination early without an error.
type listOutputCallbackWithStop func(v json.RawMessage) (stop bool, err error)

// NewClient creates a new Shippo API client instance.
func NewClient(privateToken, apiVersion string) *Client {
	return &Client{
		privateToken: privateToken,
		apiVersion:   apiVersion,
		baseURL:      shippoAPIBaseURL,
		httpClient:   newInstrumentedHTTPClient(),
	}
}

// newInstrumentedHTTPClient returns an HTTP client whose transport is wrapped
// with OpenTelemetry instrumentation so every Shippo API request emits a client
// span, giving visibility into outbound Shippo calls. Only the W3C trace
// context is propagated to Shippo; baggage is intentionally excluded so
// internal metadata is never sent to a third party. When no TracerProvider is
// configured the OTel API falls back to a no-op, so this adds negligible
// overhead for callers that do not use tracing.
func newInstrumentedHTTPClient() *http.Client {
	return &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithSpanNameFormatter(spanName),
			otelhttp.WithPropagators(propagation.TraceContext{}),
		),
	}
}

// spanName names client spans as "host METHOD /normalized/path", collapsing
// object-ID path segments to "{id}" so high-cardinality identifiers do not
// fragment span names.
func spanName(_ string, r *http.Request) string {
	return r.URL.Host + " " + r.Method + " " + normalizePathIDs(r.URL.Path)
}

// normalizePathIDs replaces Shippo object-ID path segments (all-digit segments,
// UUIDs, and long hexadecimal identifiers) with "{id}" to bound span-name
// cardinality.
func normalizePathIDs(p string) string {
	segments := strings.Split(p, "/")
	for i, s := range segments {
		if isObjectID(s) {
			segments[i] = "{id}"
		}
	}
	return strings.Join(segments, "/")
}

// isObjectID reports whether a path segment looks like a Shippo identifier.
// Shippo object IDs are 32-character hex strings and UUIDs are 36-character
// hyphenated hex; any sufficiently long hex-only segment is treated as an ID.
func isObjectID(s string) bool {
	if isAllDigits(s) {
		return true
	}
	hexLen := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
			hexLen++
		case r == '-':
		default:
			return false
		}
	}
	return hexLen >= 16
}

// isAllDigits reports whether s is non-empty and contains only ASCII digits.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// SetTraceLogger sets a new trace logger and returns the old logger.
// If logger is not nil, Client will output all internal messages to the logger.
func (c *Client) SetTraceLogger(logger *log.Logger) *log.Logger {
	oldLogger := c.logger
	c.logger = logger
	return oldLogger
}

func (c *Client) do(ctx context.Context, method, path string, input, output interface{}) error {
	url := c.baseURL + path

	req, err := c.createRequest(ctx, method, url, input)
	if err != nil {
		return fmt.Errorf("error creating request object: %s", err.Error())
	}

	if err := c.executeRequest(req, output); err != nil {
		if aerr, ok := err.(*errors.APIError); ok {
			return aerr
		}
		return fmt.Errorf("error executing request: %s", err.Error())
	}

	return nil
}

func (c *Client) doList(ctx context.Context, method, path string, input interface{}, outputCallback listOutputCallback) error {
	nextURL := c.baseURL + path + "?results=25"

	for {
		req, err := c.createRequest(ctx, method, nextURL, input)
		if err != nil {
			return fmt.Errorf("error creating request object: %s", err.Error())
		}

		listOutput := &models.ListAPIOutput{}
		if err := c.executeRequest(req, listOutput); err != nil {
			if aerr, ok := err.(*errors.APIError); ok {
				return aerr
			}
			return fmt.Errorf("error executing request: %s", err.Error())
		}

		for _, v := range listOutput.Results {
			if err := outputCallback(v); err != nil {
				return fmt.Errorf("error unmarshalling output item: %s", err.Error())
			}
		}

		if listOutput.NextPageURL == nil {
			break
		}

		nextURL = *listOutput.NextPageURL
	}

	return nil
}

func (c *Client) doListWithStop(ctx context.Context, method, path string, input interface{}, outputCallback listOutputCallbackWithStop) error {
	nextURL := c.baseURL + path + "?results=100"

	for {
		req, err := c.createRequest(ctx, method, nextURL, input)
		if err != nil {
			return fmt.Errorf("error creating request object: %s", err.Error())
		}

		listOutput := &models.ListAPIOutput{}
		if err := c.executeRequest(req, listOutput); err != nil {
			if aerr, ok := err.(*errors.APIError); ok {
				return aerr
			}
			return fmt.Errorf("error executing request: %s", err.Error())
		}

		for _, v := range listOutput.Results {
			stop, err := outputCallback(v)
			if err != nil {
				return fmt.Errorf("error unmarshalling output item: %s", err.Error())
			}
			if stop {
				return nil
			}
		}

		if listOutput.NextPageURL == nil {
			break
		}

		nextURL = *listOutput.NextPageURL
	}

	return nil
}

func (c *Client) createRequest(ctx context.Context, method, url string, bodyObject interface{}) (req *http.Request, err error) {
	var reqBodyDebug []byte

	if c.logger != nil {
		defer func() {
			if err != nil {
				c.logPrintf("Client.createRequest() error: %s", err.Error())
				return
			} else if req == nil {
				c.logPrintf("Client.createRequest() req=nil, error=nil")
				return
			}

			headers := []string{}
			for hk, hva := range req.Header {
				for _, hv := range hva {
					headers = append(headers, fmt.Sprintf("%s=%s", hk, hv))
				}
			}

			body := ""
			if reqBodyDebug != nil {
				body = string(reqBodyDebug)
			}

			c.logPrintf("Client.createRequest() HTTP request created: method=%q, url=%q, headers=%q, body=%q",
				req.Method, req.URL.String(), strings.Join(headers, ","), body)
		}()
	}

	var reqBody io.Reader
	if bodyObject != nil {
		data, err := json.Marshal(bodyObject)
		if err != nil {
			return nil, fmt.Errorf("error marshaling body object: %s", err.Error())
		}

		reqBodyDebug = data

		reqBody = bytes.NewBuffer(data)
	}

	req, err = http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %s", err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "ShippoToken "+c.privateToken)
	if c.apiVersion != "" {
		req.Header.Set("Shippo-API-Version", c.apiVersion)
	}

	// no keep-alive
	req.Header.Set("Connection", "close")
	req.Close = true

	return req, nil
}

func (c *Client) executeRequest(req *http.Request, output interface{}) (err error) {
	if c.logger != nil {
		defer func() {
			if err != nil {
				c.logPrintf("Client.executeRequest() error: %s", err.Error())
			}
		}()
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making HTTP request: %s", err.Error())
	}
	defer res.Body.Close()

	resData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading response body data: %s", err.Error())
	}

	if c.logger != nil {
		c.logPrintf("Client.executeRequest() response: status=%q, body=%q", res.Status, string(resData))
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		if output != nil && len(resData) > 0 {
			if err := json.Unmarshal(resData, output); err != nil {
				return fmt.Errorf("error unmarshaling response data: %s", err.Error())
			}
		}

		return nil
	}

	return &errors.APIError{
		Status:       res.StatusCode,
		ResponseBody: resData,
	}
}

func (c *Client) logPrintf(format string, args ...interface{}) {
	if c.logger != nil {
		c.logger.Printf(format, args...)
	}
}
