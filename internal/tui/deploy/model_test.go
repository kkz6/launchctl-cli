package deploy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
)

func TestPollingReconcilesMissedCompletionAfterTaskDiscovery(t *testing.T) {
	for _, status := range []string{"finished", "failed", "timeout", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/servers/server-1/sites/site-1/deployments/deploy-1" {
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				_, _ = w.Write([]byte(`{"success":true,"data":{"id":"deploy-1","task_id":"task-1","status":"` + status + `"}}`))
			}))
			defer server.Close()
			client := api.NewClient(&config.Config{APIURL: server.URL})
			model := NewModel(Opts{Client: client, ServerID: "server-1", SiteID: "site-1", DeploymentID: "deploy-1"})
			defer model.Close()
			model.taskID = "task-1"
			model.status = "installing"
			updated, poll := model.Update(pollTickMsg{})
			if poll == nil {
				t.Fatal("status polling stopped after task discovery")
			}
			updated, _ = updated.Update(poll())
			final := updated.(Model)
			if !final.Done() || final.status != status || (final.Err() == nil) != (status == "finished") {
				t.Fatalf("done=%v status=%q error=%v", final.Done(), final.status, final.Err())
			}
		})
	}
}

func TestPollingCompletesDeploymentThatFinishedBeforeSubscription(t *testing.T) {
	model := NewModel(Opts{DeploymentID: "fast-deployment"})
	defer model.Close()
	updated, _ := model.Update(deploymentPolledMsg{deployment: &api.DeploymentResponse{Status: "finished"}})
	final := updated.(Model)
	if !final.Done() || final.Err() != nil {
		t.Fatalf("done=%v error=%v", final.Done(), final.Err())
	}
	updated, _ = final.Update(statusMsg{status: "installing"})
	if updated.(Model).status != "finished" {
		t.Fatal("late progress event regressed the terminal state")
	}
}

func TestDisconnectedEventsKeepRESTReconciliationActive(t *testing.T) {
	model := NewModel(Opts{DeploymentID: "deploy-1"})
	defer model.Close()
	updated, _ := model.Update(wsErrorMsg{err: errors.New("socket closed")})
	model = updated.(Model)
	if model.Done() || model.Err() != nil {
		t.Fatal("websocket failure was reported as deployment failure")
	}
	updated, next := model.Update(deploymentPolledMsg{err: errors.New("temporary connection error")})
	if next == nil || updated.(Model).Done() {
		t.Fatal("transient HTTP failure stopped polling")
	}
	updated, _ = updated.Update(deploymentPolledMsg{deployment: &api.DeploymentResponse{Status: "finished"}})
	if !updated.(Model).Done() || updated.(Model).Err() != nil {
		t.Fatal("REST completion was not reconciled")
	}
}

func TestTerminalProgressEventCompletesView(t *testing.T) {
	model := NewModel(Opts{DeploymentID: "deploy-1"})
	defer model.Close()
	updated, _ := model.Update(statusMsg{status: "timeout"})
	if !updated.(Model).Done() || updated.(Model).Err() == nil {
		t.Fatal("terminal progress status did not fail the deployment")
	}
}

func TestPermanentStatusErrorIsVisible(t *testing.T) {
	model := NewModel(Opts{DeploymentID: "deploy-1"})
	defer model.Close()
	updated, next := model.Update(deploymentPolledMsg{err: &api.APIError{StatusCode: 403, Message: "Forbidden"}})
	if !updated.(Model).Done() || updated.(Model).Err() == nil || next != nil {
		t.Fatal("permanent error did not stop monitoring")
	}
}

func TestClosingViewCancelsPolling(t *testing.T) {
	model := NewModel(Opts{})
	model.Close()
	if !errors.Is(model.ctx.Err(), context.Canceled) {
		t.Fatal("close did not cancel status requests")
	}
	_, next := model.Update(pollTickMsg{})
	if next != nil {
		t.Fatal("closed view scheduled another poll")
	}
}
