package httpclient_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go-feature-based-boilerplate/pkg/contextutil"
	"go-feature-based-boilerplate/pkg/httpclient"
	"go-feature-based-boilerplate/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestClient_HTTPMethods(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("got"))
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("posted:" + string(body)))
		case http.MethodPut:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("putted"))
		case http.MethodPatch:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("patched"))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case httpclient.MethodQuery:
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("queried:" + string(body)))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer ts.Close()

	client := httpclient.New(httpclient.WithTimeout(5 * time.Second))

	ctx := context.Background()

	// 1. GET
	resp, err := client.Get(ctx, ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "got", string(body))

	// 2. POST
	resp, err = client.Post(ctx, ts.URL, "text/plain", strings.NewReader("hello"))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "posted:hello", string(body))

	// 3. PUT
	resp, err = client.Put(ctx, ts.URL, "text/plain", strings.NewReader("data"))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 4. PATCH
	resp, err = client.Patch(ctx, ts.URL, "text/plain", strings.NewReader("patch"))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 5. DELETE
	resp, err = client.Delete(ctx, ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// 6. QUERY (IETF RFC 10008)
	resp, err = client.Query(ctx, ts.URL, "text/plain", strings.NewReader("search-query"))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "queried:search-query", string(body))
}

func TestClient_QueryAndQueryJSON(t *testing.T) {
	type SearchFilter struct {
		Keyword string `json:"keyword"`
		Limit   int    `json:"limit"`
	}
	type SearchResult struct {
		Hits []string `json:"hits"`
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, httpclient.MethodQuery, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		var filter SearchFilter
		_ = json.NewDecoder(r.Body).Decode(&filter)
		assert.Equal(t, "golang microservice", filter.Keyword)

		resp := SearchResult{Hits: []string{"article-1", "article-2"}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := httpclient.New()
	ctx := context.Background()

	var result SearchResult
	filter := SearchFilter{Keyword: "golang microservice", Limit: 10}

	resp, err := client.QueryJSON(ctx, ts.URL, filter, &result)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, result.Hits, 2)
	assert.Equal(t, "article-1", result.Hits[0])
}

func TestClient_JSONHelpers(t *testing.T) {
	type UserDTO struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		if r.Method == http.MethodGet {
			resp := UserDTO{ID: 101, Name: "John Doe", Email: "john@example.com"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodPost {
			var incoming UserDTO
			_ = json.NewDecoder(r.Body).Decode(&incoming)
			incoming.ID = 999
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(incoming)
			return
		}
	}))
	defer ts.Close()

	client := httpclient.New()
	ctx := context.Background()

	// Test GetJSON
	var user UserDTO
	resp, err := client.GetJSON(ctx, ts.URL, &user)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 101, user.ID)
	assert.Equal(t, "John Doe", user.Name)

	// Test PostJSON
	input := UserDTO{Name: "Alice", Email: "alice@example.com"}
	var created UserDTO
	resp, err = client.PostJSON(ctx, ts.URL, input, &created)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 999, created.ID)
	assert.Equal(t, "Alice", created.Name)
}

func TestClient_HeadersAndQueryParameters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "custom-header-val", r.Header.Get("X-Custom-Header"))
		assert.Equal(t, "default-val", r.Header.Get("X-Default-Header"))
		assert.Equal(t, "asc", r.URL.Query().Get("sort"))
		assert.Equal(t, "25", r.URL.Query().Get("limit"))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := httpclient.New(
		httpclient.WithDefaultHeader("X-Default-Header", "default-val"),
	)

	resp, err := client.Get(context.Background(), ts.URL,
		httpclient.WithRequestHeader("X-Custom-Header", "custom-header-val"),
		httpclient.WithQueryParam("sort", "asc"),
		httpclient.WithQueryParam("limit", "25"),
	)

	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_TraceAndRequestIDPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Check Request-ID header
		assert.Equal(t, "req-upstream-12345", r.Header.Get("X-Request-ID"))

		// 2. Check W3C traceparent header
		traceparent := r.Header.Get("traceparent")
		assert.NotEmpty(t, traceparent)
		assert.Contains(t, traceparent, "4bf92f3577b34da6a3ce929d0e0e4736")

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := httpclient.New()

	// Prepare context with RequestID and OpenTelemetry TraceContext
	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)
	ctx = contextutil.WithRequestID(ctx, "req-upstream-12345")

	resp, err := client.Get(ctx, ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_RetryOnTransientFailure(t *testing.T) {
	var attempts int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("recovered"))
	}))
	defer ts.Close()

	client := httpclient.New(
		httpclient.WithRetry(3, 10*time.Millisecond, 50*time.Millisecond),
	)

	resp, err := client.Get(context.Background(), ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_ContextTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := httpclient.New(httpclient.WithTimeout(50 * time.Millisecond))

	_, err := client.Get(context.Background(), ts.URL)
	require.Error(t, err)
}

func TestClient_LoggingIntegration(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	zapLogger := zap.New(core)
	appLogger := logger.NewZapLogger(zapLogger, "test-service", "test")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := httpclient.New(httpclient.WithLogger(appLogger))

	resp, err := client.Get(context.Background(), ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	entries := logs.All()
	require.Len(t, entries, 1)
	fields := entries[0].ContextMap()
	assert.Equal(t, http.MethodGet, fields["http.client.method"])
	assert.Equal(t, int64(200), fields["http.client.status_code"])
	assert.NotEmpty(t, fields["http.client.duration_ms"])
}

func TestClient_StandardClientAccess(t *testing.T) {
	client := httpclient.New(httpclient.WithTimeout(3 * time.Second))
	std := client.StandardClient()
	require.NotNil(t, std)
	assert.Equal(t, 3*time.Second, std.Timeout)
}
