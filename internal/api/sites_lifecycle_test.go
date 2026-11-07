package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/config"
)

func TestSiteLifecycleRequests(t *testing.T) {
	empty := ""
	queue := false
	for _, test := range []struct {
		name, method, path, body, data string
		call                           func(*Client) error
	}{
		{
			"create", http.MethodPost, "/servers/server-1/sites/",
			`{"address":"app.example.com","type":"laravel","php_version":"php84","zero_downtime_deployment":false}`,
			`{"id":"site-1","address":"app.example.com","status":"installing"}`,
			func(c *Client) error {
				site, err := c.CreateSiteContext(context.Background(), "server-1", CreateSiteRequest{Address: "app.example.com", Type: "laravel", PHPVersion: "php84"})
				if err == nil && (site.ID != "site-1" || site.Status != "installing") {
					t.Errorf("created site = %#v", site)
				}
				return err
			},
		},
		{
			"update", http.MethodPut, "/servers/server-1/sites/site-1",
			`{"web_folder":"","queue_deployments":false}`,
			`{"id":"site-1","php_version":"php83","pending_php_version":"php84","pending_caddyfile_update_since":"2026-11-07T12:00:00Z","queue_deployments":false}`,
			func(c *Client) error {
				site, err := c.UpdateSite(context.Background(), "server-1", "site-1", UpdateSiteRequest{WebFolder: &empty, QueueDeployments: &queue})
				if err == nil && (site.PendingPHPVersion != "php84" || site.PendingCaddyfileUpdateSince == nil || site.QueueDeployments) {
					t.Errorf("updated site = %#v", site)
				}
				return err
			},
		},
		{
			"delete", http.MethodDelete, "/servers/server-1/sites/site-1", "", "null",
			func(c *Client) error { return c.DeleteSite(context.Background(), "server-1", "site-1") },
		},
		{
			"enable", http.MethodPost, "/servers/server-1/sites/site-1/auto-deployment/enable", "", "null",
			func(c *Client) error {
				return c.SetSiteAutoDeployment(context.Background(), "server-1", "site-1", true)
			},
		},
		{
			"disable", http.MethodPost, "/servers/server-1/sites/site-1/auto-deployment/disable", "", "null",
			func(c *Client) error {
				return c.SetSiteAutoDeployment(context.Background(), "server-1", "site-1", false)
			},
		},
	} {
		for _, prefix := range []string{"", "/api"} {
			t.Run(test.name+prefix, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != test.method || r.URL.Path != prefix+test.path {
						t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, test.method, prefix+test.path)
					}
					if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("X-Team-ID") != "team-1" {
						t.Error("missing authentication or team scope")
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
					}
					if test.body == "" {
						if len(body) != 0 {
							t.Errorf("unexpected body = %s", body)
						}
					} else {
						var got, want any
						if err := json.Unmarshal(body, &got); err != nil {
							t.Error(err)
						}
						_ = json.Unmarshal([]byte(test.body), &want)
						if !reflect.DeepEqual(got, want) {
							t.Errorf("body = %s, want %s", body, test.body)
						}
					}
					_, _ = io.WriteString(w, `{"success":true,"data":`+test.data+`}`)
				}))
				defer server.Close()
				client := NewClient(&config.Config{APIURL: server.URL + prefix, AccessToken: "test-token", TeamID: "team-1"})
				if err := test.call(client); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestSiteLifecycleErrorsAndContext(t *testing.T) {
	calls := map[string]func(context.Context, *Client) error{
		"create": func(ctx context.Context, c *Client) error {
			_, err := c.CreateSiteContext(ctx, "server-1", CreateSiteRequest{})
			return err
		},
		"update": func(ctx context.Context, c *Client) error {
			_, err := c.UpdateSite(ctx, "server-1", "site-1", UpdateSiteRequest{})
			return err
		},
		"delete": func(ctx context.Context, c *Client) error { return c.DeleteSite(ctx, "server-1", "site-1") },
		"auto-deploy": func(ctx context.Context, c *Client) error {
			return c.SetSiteAutoDeployment(ctx, "server-1", "site-1", true)
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = io.WriteString(w, `{"success":false,"message":"unavailable"}`)
			}))
			defer server.Close()
			client := NewClient(&config.Config{APIURL: server.URL})
			var apiErr *APIError
			if err := call(context.Background(), client); !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("error = %v, want service unavailable", err)
			}
			if requests.Load() != 1 {
				t.Fatalf("mutation retried: %d requests", requests.Load())
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := call(ctx, client); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context cancellation", err)
			}
			if requests.Load() != 1 {
				t.Fatal("cancelled call sent a request")
			}
		})
	}
}

func TestUpdateSiteCancelsInFlightRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := NewClient(&config.Config{APIURL: server.URL}).UpdateSite(ctx, "s", "site", UpdateSiteRequest{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
}
