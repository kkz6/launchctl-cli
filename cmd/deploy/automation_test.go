package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/spf13/cobra"
)

func deploymentTestCommand(cmd *cobra.Command) (*cobra.Command, *bytes.Buffer) {
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().Bool("ci", false, "")
	cmd.SilenceErrors = true
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	return cmd, out
}

func deploymentTestConfig(t *testing.T, url string) {
	t.Helper()
	previous := appstate.GetConfig()
	appstate.SetConfig(&config.Config{APIURL: url, AccessToken: "test-token", TeamID: "team-1"})
	t.Cleanup(func() { appstate.SetConfig(previous) })
}

func TestTriggerUsesProjectDefaultsAndJSONWithoutOpeningLiveView(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := config.SaveProject(&config.ProjectConfig{Server: "default-server", Site: "default-site"}); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		args []string
		path string
	}{
		{"defaults", []string{"--json"}, "/servers/default-server/sites/default-site/deploy"},
		{"explicit", []string{"site-1", "--server", "server-1", "--json"}, "/servers/server-1/sites/site-1/deploy"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != test.path {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("X-Team-ID") != "team-1" {
					t.Error("missing authentication or team header")
				}
				fmt.Fprint(w, `{"success":true,"data":{"id":"new-deployment","status":"pending"}}`)
			}))
			defer server.Close()
			deploymentTestConfig(t, server.URL)
			// Root resolves --api-url after env overrides. The command must not overwrite it again.
			t.Setenv("LAUNCHCTL_API_URL", "http://127.0.0.1:1")
			cmd, out := deploymentTestCommand(newTriggerCommand())
			cmd.SetArgs(test.args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var result api.DeploymentResponse
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON %q: %v", out.String(), err)
			}
			if result.ID != "new-deployment" || requests.Load() != 1 {
				t.Fatalf("result = %#v, requests = %d", result, requests.Load())
			}
		})
	}
}

func TestDeploymentCommandsWaitWithoutCIAndReportTerminalResults(t *testing.T) {
	for _, action := range []string{"trigger", "rollback"} {
		for _, status := range []string{"finished", "failed", "timeout"} {
			t.Run(action+"/"+status, func(t *testing.T) {
				var posts, gets atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPost {
						posts.Add(1)
						want := "/servers/server-1/sites/site-1/deploy"
						if action == "rollback" {
							want = "/servers/server-1/sites/site-1/rollback/old-deployment"
						}
						if r.URL.Path != want {
							t.Errorf("mutation path = %s, want %s", r.URL.Path, want)
						}
						fmt.Fprint(w, `{"success":true,"data":{"id":"new-deployment","status":"pending"}}`)
						return
					}
					gets.Add(1)
					if r.Method != http.MethodGet || r.URL.Path != "/servers/server-1/sites/site-1/deployments/new-deployment" {
						t.Errorf("status request = %s %s", r.Method, r.URL.Path)
					}
					fmt.Fprintf(w, `{"success":true,"data":{"id":"new-deployment","status":%q}}`, status)
				}))
				defer server.Close()
				deploymentTestConfig(t, server.URL)
				command, args := newTriggerCommand(), []string{"site-1"}
				if action == "rollback" {
					command, args = newRollbackCommand(), []string{"old-deployment", "--site", "site-1", "--yes"}
				}
				cmd, out := deploymentTestCommand(command)
				cmd.SetArgs(append(args, "--server", "server-1", "--wait", "--json"))
				err := cmd.Execute()
				if status == "finished" && err != nil || status != "finished" && (err == nil || !strings.Contains(err.Error(), status)) {
					t.Fatalf("status %s: error = %v", status, err)
				}
				var result api.DeploymentResponse
				if err := json.Unmarshal(out.Bytes(), &result); err != nil {
					t.Fatalf("invalid JSON %q: %v", out.String(), err)
				}
				if result.Status != status || posts.Load() != 1 || gets.Load() != 1 {
					t.Fatalf("result = %#v, requests = %d POST + %d GET", result, posts.Load(), gets.Load())
				}
			})
		}
	}
}

func TestRollbackRequiresExplicitConfirmationInAutomation(t *testing.T) {
	for _, mode := range []string{"--json", "--ci"} {
		t.Run(mode, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
			defer server.Close()
			deploymentTestConfig(t, server.URL)
			cmd, out := deploymentTestCommand(newRollbackCommand())
			cmd.SetArgs([]string{"old-deployment", "--server", "s", "--site", "site", mode})
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--yes") {
				t.Fatalf("error = %v", err)
			}
			if requests.Load() != 0 || out.Len() != 0 {
				t.Fatalf("unconfirmed rollback sent %d requests or output %q", requests.Load(), out.String())
			}
		})
	}
}

func TestInvalidWaitTimeoutDoesNotTriggerMutation(t *testing.T) {
	for _, constructor := range []func() *cobra.Command{newTriggerCommand, newRollbackCommand} {
		cmd, _ := deploymentTestCommand(constructor())
		cmd.SetArgs([]string{"id", "--wait", "--timeout", "0"})
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--timeout") {
			t.Fatalf("error = %v", err)
		}
	}
}

func TestWaitForDeploymentCancelsInFlightRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	client := api.NewClient(&config.Config{APIURL: server.URL})
	started := time.Now()
	_, err := waitForDeployment(context.Background(), client, "s", "site", "d", 40*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("wait exceeded its deadline")
	}
}

func TestWaitForDeploymentCancelsBetweenPolls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"data":{"id":"d","status":"queued"}}`)
	}))
	defer server.Close()
	time.AfterFunc(40*time.Millisecond, cancel)
	client := api.NewClient(&config.Config{APIURL: server.URL})
	started := time.Now()
	result, err := waitForDeployment(ctx, client, "s", "site", "d", time.Minute)
	if !errors.Is(err, context.Canceled) || result == nil || result.Status != "queued" {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("cancellation waited for the poll interval")
	}
}
