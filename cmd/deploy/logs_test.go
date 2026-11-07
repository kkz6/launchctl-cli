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
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/spf13/cobra"
)

func logsTestCommand(jsonOutput bool) (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", jsonOutput, "")
	cmd.Flags().Bool("ci", true, "")
	cmd.SetContext(context.Background())
	var output bytes.Buffer
	cmd.SetOut(&output)
	return cmd, &output
}

func TestStoredDeploymentLogsFetchExactTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers/server-1/tasks/old-task" {
			t.Errorf("unexpected request %s: should fetch the exact task, not the latest task list", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":"old-task","server_id":"server-1","output":"old deployment output\n"}}`))
	}))
	defer server.Close()
	client := api.NewClient(&config.Config{APIURL: server.URL})
	cmd, output := logsTestCommand(false)
	taskID := "old-task"
	err := showStoredLogs(cmd, client, "server-1", &api.DeploymentResponse{ID: "short", TaskID: &taskID, Status: "finished"})
	if err != nil || output.String() != "old deployment output\n" {
		t.Fatalf("output=%q error=%v", output.String(), err)
	}
}

func TestStoredDeploymentLogsJSONRepresentsQueuedDeployment(t *testing.T) {
	cmd, output := logsTestCommand(true)
	err := showStoredLogs(cmd, nil, "server-1", &api.DeploymentResponse{ID: "queued-1", Status: "queued"})
	if err != nil {
		t.Fatal(err)
	}
	var snapshot deploymentLogsSnapshot
	if err := json.Unmarshal(output.Bytes(), &snapshot); err != nil {
		t.Fatalf("invalid JSON %q: %v", output.String(), err)
	}
	if snapshot.Deployment.ID != "queued-1" || snapshot.Task != nil {
		t.Fatalf("unexpected snapshot %+v", snapshot)
	}
}

func TestFollowRecognizesBackendActiveStatuses(t *testing.T) {
	for _, status := range []string{"pending", "queued", "installing", "deploying"} {
		if !deploymentLogsActive(status) {
			t.Errorf("%s must be followed", status)
		}
	}
	for _, status := range []string{"finished", "failed", "timeout", "cancelled"} {
		if deploymentLogsActive(status) {
			t.Errorf("%s must be terminal", status)
		}
	}
}

func TestNoninteractiveFollowEmitsAppendedOutput(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/servers/server-1/tasks/task-1":
			requests++
			output := "building\n"
			if requests > 1 {
				output += "done\n"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"id": "task-1", "server_id": "server-1", "output": output}})
		case "/servers/server-1/sites/site-1/deployments/deploy-1":
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"deploy-1","status":"finished","task_id":"task-1"}}`))
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := api.NewClient(&config.Config{APIURL: server.URL})
	cmd, output := logsTestCommand(false)
	taskID := "task-1"
	err := followStoredLogsInterval(cmd, client, "server-1", "site-1", &api.DeploymentResponse{ID: "deploy-1", Status: "installing", TaskID: &taskID}, time.Millisecond)
	if err != nil || output.String() != "building\ndone\n" {
		t.Fatalf("output=%q error=%v", output.String(), err)
	}
}

func TestJSONFollowKeepsValidNDJSONOnTerminalFailure(t *testing.T) {
	for _, status := range []string{"failed", "timeout", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/tasks/task-1") {
					_, _ = w.Write([]byte(`{"success":true,"data":{"id":"task-1","server_id":"server-1","output":"deploy error\n"}}`))
					return
				}
				_, _ = fmt.Fprintf(w, `{"success":true,"data":{"id":"deploy-1","status":%q,"task_id":"task-1"}}`, status)
			}))
			defer server.Close()
			client := api.NewClient(&config.Config{APIURL: server.URL})
			cmd, output := logsTestCommand(true)
			err := followStoredLogsInterval(cmd, client, "server-1", "site-1", &api.DeploymentResponse{ID: "deploy-1", Status: "queued"}, time.Millisecond)
			if err == nil || !strings.Contains(err.Error(), status) {
				t.Fatalf("expected terminal error, got %v", err)
			}
			lines := strings.Split(strings.TrimSpace(output.String()), "\n")
			if len(lines) != 2 {
				t.Fatalf("expected two snapshots, got %q", output.String())
			}
			for _, line := range lines {
				if !json.Valid([]byte(line)) {
					t.Fatalf("invalid NDJSON: %q", line)
				}
			}
			var final deploymentLogsSnapshot
			_ = json.Unmarshal([]byte(lines[1]), &final)
			if final.Deployment.Status != status || final.output() != "deploy error\n" {
				t.Fatalf("unexpected final snapshot %+v", final)
			}
		})
	}
}

func TestNoninteractiveFollowIsCancellable(t *testing.T) {
	cmd, output := logsTestCommand(true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)
	err := followStoredLogsInterval(cmd, nil, "server-1", "site-1", &api.DeploymentResponse{ID: "deploy-1", Status: "queued"}, time.Hour)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v, want cancellation", err)
	}
	if !json.Valid(bytes.TrimSpace(output.Bytes())) {
		t.Fatalf("invalid JSON snapshot on cancellation: %q", output.String())
	}
}

func TestLogsFailureDoesNotPrintUsage(t *testing.T) {
	if !logsCmd.SilenceUsage {
		t.Fatal("log errors would append command usage to machine-readable output")
	}
}

func TestFollowReportsFailureAlreadyTerminalAtFirstRead(t *testing.T) {
	for _, status := range []string{"failed", "timeout", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/servers/server-1/sites/site-1":
					_, _ = w.Write([]byte(`{"success":true,"data":{"id":"site-1","address":"example.test"}}`))
				case "/servers/server-1/sites/site-1/deployments/deploy-1":
					_, _ = fmt.Fprintf(w, `{"success":true,"data":{"id":"deploy-1","status":%q}}`, status)
				default:
					t.Errorf("unexpected endpoint %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			oldConfig := appstate.GetConfig()
			oldServer, oldFollow := logsServerFlag, logsFollowFlag
			t.Cleanup(func() {
				appstate.SetConfig(oldConfig)
				logsServerFlag, logsFollowFlag = oldServer, oldFollow
			})
			appstate.SetConfig(&config.Config{APIURL: server.URL})
			logsServerFlag, logsFollowFlag = "server-1", true
			cmd, output := logsTestCommand(true)
			err := logsCmd.RunE(cmd, []string{"site-1", "deploy-1"})
			if err == nil || !strings.Contains(err.Error(), status) || !json.Valid(bytes.TrimSpace(output.Bytes())) {
				t.Fatalf("output=%q error=%v", output.String(), err)
			}
		})
	}
}

func TestStoredLogsRejectMismatchedTaskResponse(t *testing.T) {
	for _, mismatch := range []string{"server", "task"} {
		t.Run(mismatch, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, serverID := "task-1", "server-1"
				if mismatch == "server" {
					serverID = "other-server"
				} else {
					id = "other-task"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"id": id, "server_id": serverID, "output": "must not be displayed"}})
			}))
			defer server.Close()
			client := api.NewClient(&config.Config{APIURL: server.URL})
			cmd, output := logsTestCommand(true)
			taskID := "task-1"
			err := showStoredLogs(cmd, client, "server-1", &api.DeploymentResponse{TaskID: &taskID})
			if err == nil || output.Len() != 0 {
				t.Fatalf("output=%q error=%v", output.String(), err)
			}
		})
	}
}
