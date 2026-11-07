package servers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/spf13/cobra"
)

func serverTestCommand(t *testing.T, cmd *cobra.Command, apiURL string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	previous := appstate.GetConfig()
	appstate.SetConfig(&config.Config{APIURL: apiURL, AccessToken: "test-token", TeamID: "team-1"})
	t.Cleanup(func() { appstate.SetConfig(previous) })
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().Bool("ci", false, "")
	cmd.SilenceErrors = true
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	return cmd, out
}

func TestCreateServerSendsSupportedCustomAndCloudSettings(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		body map[string]any
	}{
		{
			name: "custom",
			args: []string{"--provider", "custom_server", "--ip", "192.0.2.10", "--ssh-port", "2222", "--ssh-user", "admin", "--install-agent=false"},
			body: map[string]any{"provider": "custom_server", "ip": "192.0.2.10", "port": float64(2222), "ssh_user": "admin", "install_agent": false},
		},
		{
			name: "cloud",
			args: []string{"--provider", "hetzner", "--credential-id", "credential-1", "--region", "fsn1", "--size", "cx22", "--os", "ubuntu_24", "--ssh-key", "key-1,key-2"},
			body: map[string]any{"provider": "hetzner", "credential_id": "credential-1", "region": "fsn1", "size": "cx22", "operating_system": "ubuntu_24", "ssh_key_ids": []any{"key-1", "key-2"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			test.body["name"], test.body["type"] = "Server one", "docker"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/servers/" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, test.body) {
					t.Errorf("body = %#v, want %#v, error = %v", body, test.body, err)
				}
				fmt.Fprint(w, `{"success":true,"data":{"id":"server-1","name":"Server one","status":"awaiting_connection"}}`)
			}))
			defer server.Close()
			cmd, out := serverTestCommand(t, newCreateCommand(), server.URL)
			cmd.SetArgs(append(test.args, "--name", "Server one", "--type", "docker", "--yes", "--json"))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var result api.ServerResponse
			if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.ID != "server-1" || result.Status != "awaiting_connection" || requests.Load() != 1 {
				t.Fatalf("result = %q, error = %v, requests = %d", out.String(), err, requests.Load())
			}
		})
	}
}

func TestServerUpdatePreservesExplicitFalseAndEmptyDescription(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := config.SaveProject(&config.ProjectConfig{Server: "default-server"}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/servers/default-server" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		want := map[string]any{"description": "", "monitoring_enabled": false, "auto_update": false, "ssh_port": float64(2222)}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, want) {
			t.Errorf("body = %#v, want %#v, error = %v", body, want, err)
		}
		fmt.Fprint(w, `{"success":true,"data":{"id":"default-server","name":"Server one","ssh_port":2222}}`)
	}))
	defer server.Close()
	cmd, out := serverTestCommand(t, newUpdateCommand(), server.URL)
	cmd.SetArgs([]string{"--description=", "--monitoring=false", "--auto-update=false", "--ssh-port", "2222", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var result api.ServerResponse
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.SSHPort != 2222 {
		t.Fatalf("result = %q, error = %v", out.String(), err)
	}
}

func TestServerManagementRejectsInvalidOrUnconfirmedChangesBeforeRequest(t *testing.T) {
	for _, test := range []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{"missing cloud settings", newCreateCommand(), []string{"--name", "Server", "--type", "php", "--provider", "hetzner", "--yes"}},
		{"bad custom IP", newCreateCommand(), []string{"--name", "Server", "--type", "docker", "--provider", "custom_server", "--ip", "invalid", "--yes"}},
		{"unconfirmed create", newCreateCommand(), []string{"--name", "Server", "--type", "docker", "--provider", "custom_server", "--ip", "192.0.2.1"}},
		{"empty update", newUpdateCommand(), []string{"server-1"}},
		{"invalid SSH port", newUpdateCommand(), []string{"server-1", "--ssh-port", "0"}},
		{"unconfirmed reboot", newRebootCommand(), []string{"server-1"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
			defer server.Close()
			cmd, out := serverTestCommand(t, test.cmd, server.URL)
			cmd.SetArgs(append(test.args, "--json", "--ci"))
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected validation/confirmation error")
			}
			if requests.Load() != 0 || out.Len() != 0 {
				t.Fatalf("unexpected requests = %d, output = %q", requests.Load(), out.String())
			}
		})
	}
}

func TestServerLifecycleReceiptsDistinguishQueuedAndCompleted(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := config.SaveProject(&config.ProjectConfig{Server: "default-server"}); err != nil {
		t.Fatal(err)
	}
	for _, action := range lifecycleActions {
		t.Run(action.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/servers/default-server/"+action.name {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				fmt.Fprint(w, `{"success":true,"data":null}`)
			}))
			defer server.Close()
			cmd, out := serverTestCommand(t, newServerActionCommand(action), server.URL)
			args := []string{"--json"}
			if action.confirm {
				args = append(args, "--yes")
			}
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var result serverActionReceipt
			if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.ServerID != "default-server" || result.Status != action.status || result.Action != action.name {
				t.Fatalf("result = %q, error = %v", out.String(), err)
			}
		})
	}
}

func TestRebootHTTPFailureDoesNotPrintSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"success":false,"message":"permission denied"}`)
	}))
	defer server.Close()
	cmd, out := serverTestCommand(t, newRebootCommand(), server.URL)
	cmd.SetArgs([]string{"server-1", "--yes", "--json"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %v, want permission denied", err)
	}
	if out.Len() != 0 {
		t.Fatalf("unexpected success output: %q", out.String())
	}
}
