package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/spf13/cobra"
)

const composeTestPath = "/servers/server-1/docker/projects/project-1/composes"

func composeTestCommand(t *testing.T, handler http.HandlerFunc, args ...string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/servers/server-1" {
			_, _ = io.WriteString(w, `{"success":true,"data":{"id":"server-1","type":"docker"}}`)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	appstate.SetConfig(&config.Config{APIURL: server.URL})
	t.Cleanup(func() { appstate.SetConfig(nil) })
	cmd := newDockerCommand()
	cmd.PersistentFlags().Bool("json", false, "")
	cmd.PersistentFlags().Bool("ci", false, "")
	cmd.SilenceErrors = true
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(append([]string{"composes", "--server", "server-1", "--project", "project-1"}, args...))
	return cmd, stdout
}

func TestComposeCreateSendsExactFileWithoutDeployingOrPrintingSecrets(t *testing.T) {
	const contents = "services:\n  web:\n    image: nginx:1.27\n    environment:\n      SECRET: private-value\n"
	path := filepath.Join(t.TempDir(), "compose.yml")
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	cmd, stdout := composeTestCommand(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.URL.Path != composeTestPath {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var req api.CreateDockerComposeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.Name != "web" || req.ComposeSourceType != "raw_yaml" || req.RawYAML == nil || req.RawYAML.Contents != contents {
			t.Errorf("wrong create payload: %#v", req)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"id": "compose-1", "status": "idle", "raw_yaml": contents}})
	}, "create", "--name", "web", "--file", path, "--json")
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want only create", calls.Load())
	}
	var got api.DockerComposeResponse
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "compose-1" || got.Status != "idle" || got.RawYAML == nil || *got.RawYAML != "REDACTED" {
		t.Fatalf("output = %s", stdout)
	}
	if strings.Contains(stdout.String(), "private-value") {
		t.Fatal("create output leaked file contents")
	}
}

func TestComposeUpdateDistinguishesOmittedAndClearedSettings(t *testing.T) {
	emptyFile := filepath.Join(t.TempDir(), "empty.env")
	if err := os.WriteFile(emptyFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	cmd, stdout := composeTestCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != composeTestPath+"/compose-1" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body) != 2 || body["env_file"] != "" || body["run_command"] != "" {
			t.Errorf("payload = %#v", body)
		}
		_, _ = io.WriteString(w, `{"success":true,"data":{"id":"compose-1","status":"idle"}}`)
	}, "update", "compose-1", "--env-file", emptyFile, "--run-command=", "--json")
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(stdout.Bytes()) {
		t.Fatalf("output is not JSON: %s", stdout)
	}
}

func TestComposeInvalidInputsDoNotReachAPI(t *testing.T) {
	file := filepath.Join(t.TempDir(), "oversized.yml")
	if err := os.WriteFile(file, bytes.Repeat([]byte("x"), maxComposeFileBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"create", "--name", "web"}, {"create", "--name", "web", "--file", file},
		{"update", "compose-1"}, {"update", "compose-1", "--name", "  "},
		{"show", " "}, {"show", "compose-1", "--show-config"},
		{"deploy", "compose-1", "--wait", "--timeout", "0"},
		{"delete", "compose-1", "--ci"}, {"reload", "compose-1", "--json"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var calls atomic.Int32
			cmd, _ := composeTestCommand(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }, args...)
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected validation or confirmation error")
			}
			if calls.Load() != 0 {
				t.Fatalf("made %d unexpected workload requests", calls.Load())
			}
		})
	}
}

func TestComposeDeleteAndReloadReportQueuedOnly(t *testing.T) {
	for _, action := range []string{"delete", "reload"} {
		t.Run(action, func(t *testing.T) {
			cmd, stdout := composeTestCommand(t, func(w http.ResponseWriter, r *http.Request) {
				path, method := composeTestPath+"/compose-1", "DELETE"
				if action == "reload" {
					path += "/reload"
					method = "POST"
				}
				if r.Method != method || r.URL.Path != path || r.URL.RawQuery != "" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
				}
				if action == "delete" {
					w.WriteHeader(204)
				} else {
					_, _ = io.WriteString(w, `{"success":true,"data":null}`)
				}
			}, action, "compose-1", "--yes", "--json")
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			wantStatus := "queued"
			if action == "delete" {
				wantStatus = "deleting"
				if got["remove_volumes"] != false {
					t.Fatalf("volumes are not preserved: %#v", got)
				}
			}
			if got["status"] != wantStatus {
				t.Fatalf("output = %#v", got)
			}
		})
	}
}

func TestComposeDeployWaitUsesExactDeploymentAndReturnsFailureAsJSON(t *testing.T) {
	for _, status := range []string{"success", "failed", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			cmd, stdout := composeTestCommand(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case composeTestPath + "/compose-1/deploy":
					if r.Method != "POST" {
						t.Errorf("method = %s", r.Method)
					}
					w.WriteHeader(201)
					_, _ = io.WriteString(w, `{"success":true,"data":{"id":"deploy-1","server_id":"server-1","target_type":"compose","target_id":"compose-1","status":"pending"}}`)
				case composeTestPath + "/compose-1/deployments":
					fmt.Fprintf(w, `{"success":true,"data":[{"id":"other","status":"success"},{"id":"deploy-1","server_id":"server-1","target_type":"compose","target_id":"compose-1","status":%q,"error":"build failed"}]}`, status)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					w.WriteHeader(404)
				}
			}, "deploy", "compose-1", "--wait", "--json")
			err := cmd.Execute()
			if (err != nil) != (status != "success") {
				t.Fatalf("error = %v for %s", err, status)
			}
			var got api.DockerDeploymentResponse
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatalf("invalid JSON %q: %v", stdout, err)
			}
			if got.ID != "deploy-1" || got.Status != status {
				t.Fatalf("wrong deployment output: %#v", got)
			}
		})
	}
}

func TestComposeWaitRejectsWrongScopeAndTimesOutMissingDeployment(t *testing.T) {
	for _, wrongScope := range []bool{false, true} {
		t.Run(fmt.Sprint(wrongScope), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if wrongScope {
					_, _ = io.WriteString(w, `{"success":true,"data":[{"id":"deploy-1","server_id":"wrong","target_type":"compose","target_id":"compose-1","status":"success"}]}`)
				} else {
					_, _ = io.WriteString(w, `{"success":true,"data":[{"id":"other","status":"success"}]}`)
				}
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
			defer cancel()
			got, err := waitForComposeDeployment(ctx, api.NewClient(&config.Config{APIURL: server.URL}), "server-1", "project-1", "compose-1", "deploy-1")
			if got != nil || err == nil {
				t.Fatalf("got %v, %v", got, err)
			}
			if wrongScope && !strings.Contains(err.Error(), "does not match") {
				t.Fatalf("wrong error: %v", err)
			}
			if !wrongScope && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("expected timeout, got %v", err)
			}
		})
	}
}

func TestComposeShowRequiresExplicitConfigDisclosure(t *testing.T) {
	for _, showConfig := range []bool{false, true} {
		t.Run(fmt.Sprint(showConfig), func(t *testing.T) {
			args := []string{"show", "compose-1", "--json"}
			if showConfig {
				args = append(args, "--show-config")
			}
			cmd, stdout := composeTestCommand(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{"success":true,"data":{"id":"compose-1","raw_yaml":"yaml-secret","env_file":"env-secret","run_command":"cmd-secret","source_config":{"repo":"https://user:pass@example.test/repo","gha_deploy_token_hash":"token-secret"}}}`)
			}, args...)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{"yaml-secret", "env-secret", "cmd-secret"} {
				if strings.Contains(stdout.String(), secret) != showConfig {
					t.Fatalf("wrong disclosure for %q in %s", secret, stdout)
				}
			}
			if strings.Contains(stdout.String(), "token-secret") || strings.Contains(stdout.String(), "user:pass") {
				t.Fatalf("source credentials leaked: %s", stdout)
			}
		})
	}
}
