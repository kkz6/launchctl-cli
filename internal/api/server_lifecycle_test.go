package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/config"
)

func TestCreateServerOptionsMatchesBackendContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/servers/create-options" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		fmt.Fprint(w, `{"success":true,"data":{"phpVersions":{"php84":"PHP 8.4"},"databaseTypes":{"none":"None"},"serverTypes":{"docker":"Docker"},"operatingSystems":{"ubuntu_24":"Ubuntu 24.04"},"plans":{"hetzner":{"plans":[{"value":"cx22","title":"CX22"}],"regions":[{"value":"fsn1","title":"Falkenstein"}],"images":{"ubuntu_24":"ubuntu-24.04"}}},"canCreateServer":true}}`)
	}))
	defer server.Close()
	options, err := NewClient(&config.Config{APIURL: server.URL}).GetCreateServerOptionsContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !options.CanCreateServer || options.ServerTypes["docker"] != "Docker" || options.PHPVersions["php84"] != "PHP 8.4" || options.DatabaseTypes["none"] != "None" || options.OperatingSystems["ubuntu_24"] != "Ubuntu 24.04" {
		t.Fatalf("creation options lost fields: %#v", options)
	}
	provider := options.Plans["hetzner"]
	if len(provider.Plans) != 1 || provider.Plans[0].Value != "cx22" || len(provider.Regions) != 1 || provider.Regions[0].Title != "Falkenstein" || provider.Images["ubuntu_24"] != "ubuntu-24.04" {
		t.Fatalf("provider options lost fields: %#v", provider)
	}
}

func TestServerActionsCheckHTTPStatusWithoutRetrying(t *testing.T) {
	for _, action := range []struct {
		name string
		run  func(*Client, context.Context, string) error
	}{
		{"retry-provision", (*Client).RetryServerProvision},
		{"try-connection", (*Client).TryServerConnection},
		{"archive", (*Client).ArchiveServer},
		{"unarchive", (*Client).UnarchiveServer},
		{"reboot", (*Client).RebootServerContext},
	} {
		for _, status := range []int{http.StatusOK, http.StatusForbidden, http.StatusServiceUnavailable} {
			t.Run(fmt.Sprintf("%s/%d", action.name, status), func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.Method != http.MethodPost || r.URL.Path != "/servers/server-1/"+action.name {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					}
					if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("X-Team-ID") != "team-1" {
						t.Error("missing authentication or team context")
					}
					w.WriteHeader(status)
					fmt.Fprintf(w, `{"success":%t,"message":"action response","data":null}`, status == http.StatusOK)
				}))
				defer server.Close()
				client := NewClient(&config.Config{APIURL: server.URL, AccessToken: "test-token", TeamID: "team-1"})
				err := action.run(client, context.Background(), "server-1")
				if status == http.StatusOK {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					var apiErr *APIError
					if !errors.As(err, &apiErr) || apiErr.StatusCode != status {
						t.Fatalf("error = %v, want HTTP %d", err, status)
					}
				}
				if requests.Load() != 1 {
					t.Fatalf("mutation sent %d times", requests.Load())
				}
			})
		}
	}
}

func TestRebootServerPropagatesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"success":false,"message":"permission denied"}`)
	}))
	defer server.Close()
	err := NewClient(&config.Config{APIURL: server.URL}).RebootServer("server-1")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("error = %v, want HTTP 403", err)
	}
}

func TestServerActionCancelsInFlightRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := NewClient(&config.Config{APIURL: server.URL}).RetryServerProvision(ctx, "server-1")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline", err)
	}
}
