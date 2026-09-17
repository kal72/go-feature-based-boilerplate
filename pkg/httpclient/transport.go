package httpclient

import (
	"bytes"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"go-feature-based-boilerplate/pkg/contextutil"

	"go.opentelemetry.io/otel/propagation"
)

// resilientTransport is an http.RoundTripper that decorates outgoing HTTP calls
// with tracing propagation, request-id propagation, retries, and structured logging.
type resilientTransport struct {
	base       http.RoundTripper
	config     Config
	propagator propagation.TextMapPropagator
}

// newResilientTransport creates a configured RoundTripper wrapping base or http.DefaultTransport.
func newResilientTransport(cfg Config) http.RoundTripper {
	base := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &resilientTransport{
		base:       base,
		config:     cfg,
		propagator: propagation.TraceContext{},
	}
}

// RoundTrip executes a single HTTP transaction with observability and resiliency enhancements.
func (t *resilientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 1. Inject default headers
	for k, v := range t.config.DefaultHeaders {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}

	// 2. Propagate W3C traceparent and tracestate from Context
	t.propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))

	// 3. Propagate correlation Request-ID
	if req.Header.Get("X-Request-ID") == "" {
		if reqID, ok := contextutil.GetRequestID(req.Context()); ok && reqID != "" {
			req.Header.Set("X-Request-ID", reqID)
		}
	}

	// If retries are disabled, execute single roundtrip
	if t.config.RetryCount <= 0 {
		return t.executeWithLog(req)
	}

	// Buffer body if present to allow rewinding on retry
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		_ = req.Body.Close()
	}

	var resp *http.Response
	var err error

	attempts := t.config.RetryCount + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		// Rewind request body for retry
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err = t.executeWithLog(req)

		// Success or non-retryable response
		if !t.shouldRetry(resp, err) || attempt == attempts {
			break
		}

		// Close body of failed attempt before retrying to prevent resource leak
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		// Calculate exponential backoff with jitter
		delay := t.backoff(attempt)
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(delay):
		}
	}

	return resp, err
}

func (t *resilientTransport) executeWithLog(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	duration := time.Since(start)

	if t.config.Logger != nil {
		durationMs := float64(duration.Microseconds()) / 1000.0
		log := t.config.Logger.
			With("http.client.method", req.Method).
			With("http.client.url", req.URL.String()).
			With("http.client.duration_ms", durationMs)

		if resp != nil {
			log = log.With("http.client.status_code", resp.StatusCode)
		}

		if err != nil {
			log.Error(req.Context(), "outbound http request failed", err)
		} else if resp != nil && resp.StatusCode >= 400 {
			log.Warn(req.Context(), "outbound http request returned client/server error")
		} else {
			log.Debug(req.Context(), "outbound http request completed")
		}
	}

	return resp, err
}

func (t *resilientTransport) shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		// Network errors or timeout
		var netErr net.Error
		if errorsAs(err, &netErr) {
			return true
		}
		// Connection refused or reset
		if strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "connection reset") ||
			strings.Contains(err.Error(), "broken pipe") {
			return true
		}
		return false
	}

	if resp != nil {
		// Transient server errors (Bad Gateway, Service Unavailable, Gateway Timeout, Too Many Requests)
		switch resp.StatusCode {
		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, http.StatusTooManyRequests:
			return true
		}
	}

	return false
}

func (t *resilientTransport) backoff(attempt int) time.Duration {
	wait := t.config.RetryWaitMin * time.Duration(1<<uint(attempt-1))
	if wait > t.config.RetryWaitMax {
		wait = t.config.RetryWaitMax
	}
	// Add 20% jitter
	jitter := time.Duration(rand.Int63n(int64(wait) / 5))
	return wait + jitter
}

func errorsAs(err error, target any) bool {
	if err == nil {
		return false
	}
	if t, ok := target.(*net.Error); ok {
		n, ok := err.(net.Error)
		if ok {
			*t = n
			return true
		}
	}
	return false
}
