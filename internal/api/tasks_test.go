package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/config"
)

func TestGetTaskContextFetchesExactTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/servers/server-1/tasks/old-task" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" || r.Header.Get("X-Team-ID") != "team-1" {
			t.Error("missing authentication or team scope")
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":"old-task","output":"retained output"}}`))
	}))
	defer server.Close()
	client := NewClient(&config.Config{APIURL: server.URL, AccessToken: "token", TeamID: "team-1"})
	task, err := client.GetTaskContext(context.Background(), "server-1", "old-task")
	if err != nil || task == nil || task.Output == nil || *task.Output != "retained output" {
		t.Fatalf("task=%+v error=%v", task, err)
	}
}

func TestGetTaskContextPreservesNotFoundError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"message":"Task not found"}`))
	}))
	defer server.Close()
	client := NewClient(&config.Config{APIURL: server.URL})
	_, err := client.GetTaskContext(context.Background(), "server-1", "missing")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("error=%v, want API 404", err)
	}
}

func TestGetTaskContextCancelsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	client := NewClient(&config.Config{APIURL: server.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := client.GetTaskContext(ctx, "server-1", "task-1")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v, want deadline exceeded", err)
	}
}
