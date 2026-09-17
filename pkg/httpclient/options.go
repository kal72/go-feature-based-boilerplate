package httpclient

import (
	"net/http"
	"time"

	"go-feature-based-boilerplate/pkg/logger"
)

// Config defines the configuration options for the resilient HTTP client.
type Config struct {
	// Timeout specifies the maximum time an HTTP transaction can take. Default: 15s.
	Timeout time.Duration
	// MaxIdleConns controls the maximum number of idle (keep-alive) connections across all hosts. Default: 100.
	MaxIdleConns int
	// MaxIdleConnsPerHost controls the maximum idle connections per host. Default: 20.
	MaxIdleConnsPerHost int
	// MaxConnsPerHost limits the total number of connections per host. Default: 0 (unlimited).
	MaxConnsPerHost int
	// IdleConnTimeout is the maximum amount of time an idle connection will remain alive. Default: 90s.
	IdleConnTimeout time.Duration
	// TLSHandshakeTimeout specifies the maximum amount of time to wait for a TLS handshake. Default: 10s.
	TLSHandshakeTimeout time.Duration
	// RetryCount specifies the number of retries for transient errors. Default: 0 (no retries).
	RetryCount int
	// RetryWaitMin is the minimum backoff delay between retries. Default: 100ms.
	RetryWaitMin time.Duration
	// RetryWaitMax is the maximum backoff delay between retries. Default: 2s.
	RetryWaitMax time.Duration
	// DefaultHeaders holds common HTTP headers attached to every outgoing request.
	DefaultHeaders map[string]string
	// Logger is an optional structured logger for outgoing requests.
	Logger logger.Logger
}

// DefaultConfig returns a production-ready default configuration.
func DefaultConfig() Config {
	return Config{
		Timeout:             15 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		MaxConnsPerHost:     0,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		RetryCount:          0,
		RetryWaitMin:        100 * time.Millisecond,
		RetryWaitMax:        2 * time.Second,
		DefaultHeaders:      make(map[string]string),
	}
}

// Option configures the HTTP client.
type Option func(*Config)

// WithTimeout sets the request timeout duration.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		if timeout > 0 {
			c.Timeout = timeout
		}
	}
}

// WithPoolLimits sets connection pool thresholds.
func WithPoolLimits(maxIdleConns, maxIdleConnsPerHost int, idleTimeout time.Duration) Option {
	return func(c *Config) {
		if maxIdleConns > 0 {
			c.MaxIdleConns = maxIdleConns
		}
		if maxIdleConnsPerHost > 0 {
			c.MaxIdleConnsPerHost = maxIdleConnsPerHost
		}
		if idleTimeout > 0 {
			c.IdleConnTimeout = idleTimeout
		}
	}
}

// WithRetry enables automatic exponential retries for transient failures.
func WithRetry(count int, waitMin, waitMax time.Duration) Option {
	return func(c *Config) {
		if count > 0 {
			c.RetryCount = count
		}
		if waitMin > 0 {
			c.RetryWaitMin = waitMin
		}
		if waitMax > 0 {
			c.RetryWaitMax = waitMax
		}
	}
}

// WithDefaultHeader adds a default header to all outgoing requests.
func WithDefaultHeader(key, value string) Option {
	return func(c *Config) {
		if c.DefaultHeaders == nil {
			c.DefaultHeaders = make(map[string]string)
		}
		c.DefaultHeaders[key] = value
	}
}

// WithLogger configures structured logging for outbound calls.
func WithLogger(log logger.Logger) Option {
	return func(c *Config) {
		c.Logger = log
	}
}

// RequestConfig holds per-request modifications.
type RequestConfig struct {
	Headers     http.Header
	QueryParams map[string]string
}

// RequestOption modifies an individual outgoing HTTP request.
type RequestOption func(*RequestConfig)

// WithRequestHeader adds an HTTP header to this specific request.
func WithRequestHeader(key, value string) RequestOption {
	return func(rc *RequestConfig) {
		if rc.Headers == nil {
			rc.Headers = make(http.Header)
		}
		rc.Headers.Set(key, value)
	}
}

// WithQueryParam adds a query parameter to this specific request.
func WithQueryParam(key, value string) RequestOption {
	return func(rc *RequestConfig) {
		if rc.QueryParams == nil {
			rc.QueryParams = make(map[string]string)
		}
		rc.QueryParams[key] = value
	}
}
