package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/kkz6/launchctl/internal/config"
)

func TestRawRequestAddsAuthAndResolvesAPIBase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/servers" || r.URL.RawQuery != "page=2" {
			t.Fatalf("unexpected URL: %s", r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("X-Team-ID"); got != "team-a" {
			t.Fatalf("team header = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer server.Close()

	cfg := &config.Config{APIURL: server.URL + "/api", AccessToken: "secret", TeamID: "team-a"}
	client := NewClient(cfg)
	data, status, err := client.RawRequest(context.Background(), http.MethodGet, "/api/servers?page=2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || len(data) == 0 {
		t.Fatalf("status=%d body=%q", status, data)
	}
}

func TestGETRetriesTransientFailure(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClient(&config.Config{APIURL: server.URL})
	if _, _, err := client.RawRequest(context.Background(), http.MethodGet, "/api/health", nil); err != nil {
		t.Fatal(err)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
	}
}

func TestRawRequestReturnsTypedAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-ID", "req-123")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"invalid","errors":{"z":["last"],"a":["first"]}}`))
	}))
	defer server.Close()

	client := NewClient(&config.Config{APIURL: server.URL})
	_, _, err := client.RawRequest(context.Background(), http.MethodPost, "/api/test", map[string]string{"x": "y"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity || apiErr.RequestID != "req-123" {
		t.Fatalf("unexpected API error: %#v", apiErr)
	}
	if got := apiErr.Error(); got != "API request failed (422, request req-123): a: first\nz: last" {
		t.Fatalf("error text = %q", got)
	}
}

func TestResolveRequestURLRejectsInvalidOrigin(t *testing.T) {
	if _, err := resolveRequestURL("file:///tmp/api", "/api/test"); err == nil {
		t.Fatal("expected unsupported scheme error")
	}
}
