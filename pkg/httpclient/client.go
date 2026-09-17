package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// MethodQuery represents the HTTP QUERY method defined in IETF RFC 10008.
// The QUERY method provides a safe and idempotent request with a request body,
// designed specifically for rich or complex query payloads without URI length limits.
const MethodQuery = "QUERY"

// Client defines the contract for resilient outbound HTTP communications.
type Client interface {
	// Do executes an HTTP request through the resilient transport pipeline.
	Do(req *http.Request) (*http.Response, error)

	// Get issues a context-aware GET request to the target URL.
	Get(ctx context.Context, targetURL string, opts ...RequestOption) (*http.Response, error)

	// Post issues a context-aware POST request to the target URL.
	Post(ctx context.Context, targetURL string, contentType string, body io.Reader, opts ...RequestOption) (*http.Response, error)

	// Put issues a context-aware PUT request to the target URL.
	Put(ctx context.Context, targetURL string, contentType string, body io.Reader, opts ...RequestOption) (*http.Response, error)

	// Patch issues a context-aware PATCH request to the target URL.
	Patch(ctx context.Context, targetURL string, contentType string, body io.Reader, opts ...RequestOption) (*http.Response, error)

	// Delete issues a context-aware DELETE request to the target URL.
	Delete(ctx context.Context, targetURL string, opts ...RequestOption) (*http.Response, error)

	// Query issues a context-aware HTTP QUERY request (RFC 10008) carrying a request body for complex queries.
	Query(ctx context.Context, targetURL string, contentType string, body io.Reader, opts ...RequestOption) (*http.Response, error)

	// GetJSON issues a GET request and unmarshals the JSON response body into responseTarget.
	GetJSON(ctx context.Context, targetURL string, responseTarget any, opts ...RequestOption) (*http.Response, error)

	// PostJSON marshals requestBody to JSON, issues a POST request, and unmarshals the response into responseTarget.
	PostJSON(ctx context.Context, targetURL string, requestBody any, responseTarget any, opts ...RequestOption) (*http.Response, error)

	// QueryJSON marshals queryPayload to JSON, issues an HTTP QUERY request (RFC 10008), and unmarshals the response into responseTarget.
	QueryJSON(ctx context.Context, targetURL string, queryPayload any, responseTarget any, opts ...RequestOption) (*http.Response, error)

	// StandardClient returns the underlying *http.Client for integration with external SDKs.
	StandardClient() *http.Client
}

type httpClient struct {
	client *http.Client
	config Config
}

// New creates and configures a resilient Client according to the given options.
func New(opts ...Option) Client {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	transport := newResilientTransport(cfg)

	rawClient := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}

	return &httpClient{
		client: rawClient,
		config: cfg,
	}
}

// Do executes an HTTP request.
func (c *httpClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// Get issues a GET request.
func (c *httpClient) Get(ctx context.Context, targetURL string, opts ...RequestOption) (*http.Response, error) {
	return c.send(ctx, http.MethodGet, targetURL, "", nil, opts...)
}

// Post issues a POST request.
func (c *httpClient) Post(ctx context.Context, targetURL string, contentType string, body io.Reader, opts ...RequestOption) (*http.Response, error) {
	return c.send(ctx, http.MethodPost, targetURL, contentType, body, opts...)
}

// Put issues a PUT request.
func (c *httpClient) Put(ctx context.Context, targetURL string, contentType string, body io.Reader, opts ...RequestOption) (*http.Response, error) {
	return c.send(ctx, http.MethodPut, targetURL, contentType, body, opts...)
}

// Patch issues a PATCH request.
func (c *httpClient) Patch(ctx context.Context, targetURL string, contentType string, body io.Reader, opts ...RequestOption) (*http.Response, error) {
	return c.send(ctx, http.MethodPatch, targetURL, contentType, body, opts...)
}

// Delete issues a DELETE request.
func (c *httpClient) Delete(ctx context.Context, targetURL string, opts ...RequestOption) (*http.Response, error) {
	return c.send(ctx, http.MethodDelete, targetURL, "", nil, opts...)
}

// GetJSON issues a GET request and unmarshals JSON response into responseTarget.
func (c *httpClient) GetJSON(ctx context.Context, targetURL string, responseTarget any, opts ...RequestOption) (*http.Response, error) {
	opts = append(opts, WithRequestHeader("Accept", "application/json"))
	resp, err := c.Get(ctx, targetURL, opts...)
	if err != nil {
		return resp, err
	}
	defer func() { _ = resp.Body.Close() }()

	if responseTarget != nil {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return resp, fmt.Errorf("httpclient: read response body: %w", readErr)
		}
		if len(bodyBytes) > 0 {
			if unmarshalErr := json.Unmarshal(bodyBytes, responseTarget); unmarshalErr != nil {
				return resp, fmt.Errorf("httpclient: unmarshal json response: %w", unmarshalErr)
			}
		}
	}

	return resp, nil
}

// PostJSON marshals requestBody, sends POST request with Content-Type: application/json, and unmarshals response.
func (c *httpClient) PostJSON(ctx context.Context, targetURL string, requestBody any, responseTarget any, opts ...RequestOption) (*http.Response, error) {
	var bodyReader io.Reader
	if requestBody != nil {
		raw, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("httpclient: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	opts = append(opts, WithRequestHeader("Accept", "application/json"))
	resp, err := c.Post(ctx, targetURL, "application/json", bodyReader, opts...)
	if err != nil {
		return resp, err
	}
	defer func() { _ = resp.Body.Close() }()

	if responseTarget != nil {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return resp, fmt.Errorf("httpclient: read response body: %w", readErr)
		}
		if len(bodyBytes) > 0 {
			if unmarshalErr := json.Unmarshal(bodyBytes, responseTarget); unmarshalErr != nil {
				return resp, fmt.Errorf("httpclient: unmarshal json response: %w", unmarshalErr)
			}
		}
	}

	return resp, nil
}

// Query issues an HTTP QUERY request (RFC 10008) carrying a request body for complex or structured queries.
func (c *httpClient) Query(ctx context.Context, targetURL string, contentType string, body io.Reader, opts ...RequestOption) (*http.Response, error) {
	return c.send(ctx, MethodQuery, targetURL, contentType, body, opts...)
}

// QueryJSON marshals queryPayload, sends an HTTP QUERY request (RFC 10008), and unmarshals the response into responseTarget.
func (c *httpClient) QueryJSON(ctx context.Context, targetURL string, queryPayload any, responseTarget any, opts ...RequestOption) (*http.Response, error) {
	var bodyReader io.Reader
	if queryPayload != nil {
		raw, err := json.Marshal(queryPayload)
		if err != nil {
			return nil, fmt.Errorf("httpclient: marshal query payload: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	opts = append(opts, WithRequestHeader("Accept", "application/json"))
	resp, err := c.Query(ctx, targetURL, "application/json", bodyReader, opts...)
	if err != nil {
		return resp, err
	}
	defer func() { _ = resp.Body.Close() }()

	if responseTarget != nil {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return resp, fmt.Errorf("httpclient: read response body: %w", readErr)
		}
		if len(bodyBytes) > 0 {
			if unmarshalErr := json.Unmarshal(bodyBytes, responseTarget); unmarshalErr != nil {
				return resp, fmt.Errorf("httpclient: unmarshal json response: %w", unmarshalErr)
			}
		}
	}

	return resp, nil
}

// StandardClient returns the underlying *http.Client.
func (c *httpClient) StandardClient() *http.Client {
	return c.client
}

func (c *httpClient) send(ctx context.Context, method, targetURL string, contentType string, body io.Reader, opts ...RequestOption) (*http.Response, error) {
	rc := &RequestConfig{
		Headers:     make(http.Header),
		QueryParams: make(map[string]string),
	}
	for _, opt := range opts {
		opt(rc)
	}

	// Append query parameters to targetURL if present
	if len(rc.QueryParams) > 0 {
		parsedURL, err := url.Parse(targetURL)
		if err != nil {
			return nil, fmt.Errorf("httpclient: invalid url: %w", err)
		}
		query := parsedURL.Query()
		for k, v := range rc.QueryParams {
			query.Set(k, v)
		}
		parsedURL.RawQuery = query.Encode()
		targetURL = parsedURL.String()
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("httpclient: create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Apply request headers
	for k, vals := range rc.Headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	return c.Do(req)
}
