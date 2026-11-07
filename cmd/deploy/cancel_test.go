package deploy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/spf13/cobra"
)

func TestCancelQueuedUsesProjectDefaultsAndExplicitOverrides(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := config.SaveProject(&config.ProjectConfig{Server: "default-server", Site: "default-site"}); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		args []string
		path string
	}{
		{"defaults", nil, "/servers/default-server/sites/default-site/deployments/queued"},
		{"explicit", []string{"site-1", "--server", "server-1"}, "/servers/server-1/sites/site-1/deployments/queued"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodDelete || r.URL.Path != test.path {
					t.Errorf("request = %s %s, want DELETE %s", r.Method, r.URL.Path, test.path)
				}
				_, _ = io.WriteString(w, `{"success":true,"data":{"cancelled":2}}`)
			}))
			defer server.Close()
			setCancelQueuedTestConfig(t, server.URL)

			cmd, stdout := cancelQueuedTestCommand()
			cmd.SetArgs(append(append([]string{}, test.args...), "--yes", "--json", "--ci"))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var result api.CancelQueuedDeploymentsResponse
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("JSON output = %q: %v", stdout.String(), err)
			}
			if result.Cancelled != 2 || requests.Load() != 1 {
				t.Fatalf("result = %#v, requests = %d", result, requests.Load())
			}
		})
	}
}

func TestCancelQueuedRequiresYesWithoutRequests(t *testing.T) {
	for _, mode := range []string{"--ci", "--json"} {
		t.Run(mode, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
			}))
			defer server.Close()
			setCancelQueuedTestConfig(t, server.URL)

			cmd, stdout := cancelQueuedTestCommand()
			cmd.SetArgs([]string{"site-1", "--server", "server-1", mode})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "--yes") {
				t.Fatalf("error = %v, want confirmation required", err)
			}
			if requests.Load() != 0 || stdout.Len() != 0 {
				t.Fatalf("requests = %d, output = %q", requests.Load(), stdout.String())
			}
		})
	}
}

func TestCancelQueuedReportsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"success":false,"message":"permission denied"}`)
	}))
	defer server.Close()
	setCancelQueuedTestConfig(t, server.URL)

	cmd, stdout := cancelQueuedTestCommand()
	cmd.SetArgs([]string{"site-1", "--server", "server-1", "--yes", "--json"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %v, want permission denied", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected success output = %q", stdout.String())
	}
}

func setCancelQueuedTestConfig(t *testing.T, url string) {
	t.Helper()
	previous := appstate.GetConfig()
	appstate.SetConfig(&config.Config{APIURL: url})
	t.Cleanup(func() { appstate.SetConfig(previous) })
}

func cancelQueuedTestCommand() (*cobra.Command, *bytes.Buffer) {
	cmd := newCancelQueuedCommand()
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().Bool("ci", false, "")
	cmd.SilenceErrors = true
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	return cmd, stdout
}
