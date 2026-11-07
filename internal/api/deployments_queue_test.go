package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/config"
)

func TestCancelQueuedDeployments(t *testing.T) {
	for _, prefix := range []string{"", "/api"} {
		t.Run("namespace="+prefix, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || r.URL.Path != prefix+"/servers/server-1/sites/site-1/deployments/queued" {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("X-Team-ID") != "team-1" {
					t.Error("missing authentication or team scope")
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 {
					t.Errorf("request body = %q, error = %v", body, err)
				}
				_, _ = io.WriteString(w, `{"success":true,"message":"Queued deployments cancelled","data":{"cancelled":3}}`)
			}))
			defer server.Close()

			client := NewClient(&config.Config{APIURL: server.URL + prefix, AccessToken: "test-token", TeamID: "team-1"})
			result, err := client.CancelQueuedDeployments(context.Background(), "server-1", "site-1")
			if err != nil {
				t.Fatal(err)
			}
			if result.Cancelled != 3 {
				t.Fatalf("cancelled = %d, want 3", result.Cancelled)
			}
		})
	}
}

func TestCancelQueuedDeploymentsDoesNotRetryMutation(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"success":false,"message":"temporarily unavailable"}`)
	}))
	defer server.Close()

	client := NewClient(&config.Config{APIURL: server.URL})
	_, err := client.CancelQueuedDeployments(context.Background(), "server-1", "site-1")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("error = %v, want service unavailable API error", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestCancelQueuedDeploymentsContextCancelsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	client := NewClient(&config.Config{APIURL: server.URL})
	_, err := client.CancelQueuedDeployments(ctx, "server-1", "site-1")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
}
