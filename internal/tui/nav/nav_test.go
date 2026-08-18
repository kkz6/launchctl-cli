package nav

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
)

func TestServerActionOptionsAreResourceAware(t *testing.T) {
	tests := []struct {
		name       string
		serverType string
		want       serverAction
		dontWant   serverAction
	}{
		{name: "docker", serverType: "docker", want: serverActionProjects, dontWant: serverActionSites},
		{name: "docker case insensitive", serverType: "Docker", want: serverActionProjects, dontWant: serverActionSites},
		{name: "application", serverType: "application", want: serverActionSites, dontWant: serverActionProjects},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := serverActionOptions(api.ServerResponse{Type: tt.serverType})
			keys := make([]serverAction, 0, len(options))
			for _, option := range options {
				keys = append(keys, option.key)
			}
			if !slices.Contains(keys, tt.want) {
				t.Fatalf("actions %v do not contain %q", keys, tt.want)
			}
			if slices.Contains(keys, tt.dontWant) {
				t.Fatalf("actions %v unexpectedly contain %q", keys, tt.dontWant)
			}
		})
	}
}

func TestServerResourceSummaryUsesTheResourceForTheServerType(t *testing.T) {
	tests := []struct {
		name   string
		server api.ServerResponse
		want   string
	}{
		{
			name: "Docker projects",
			server: api.ServerResponse{
				Type:          "docker",
				SitesCount:    0,
				ProjectsCount: 2,
			},
			want: "2 projects",
		},
		{
			name: "Docker type is case insensitive",
			server: api.ServerResponse{
				Type:          "Docker",
				SitesCount:    5,
				ProjectsCount: 3,
			},
			want: "3 projects",
		},
		{
			name: "PHP sites",
			server: api.ServerResponse{
				Type:          "php",
				SitesCount:    1,
				ProjectsCount: 9,
			},
			want: "1 site",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serverResourceSummary(tt.server); got != tt.want {
				t.Fatalf("serverResourceSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDockerServerActionsHidePHPOnlyOperations(t *testing.T) {
	options := serverActionOptions(api.ServerResponse{Type: "docker"})
	keys := make([]serverAction, 0, len(options))
	for _, option := range options {
		keys = append(keys, option.key)
	}
	for _, excluded := range []serverAction{serverActionSites, serverActionDatabases, serverActionCron, serverActionDaemons} {
		if slices.Contains(keys, excluded) {
			t.Errorf("Docker server actions %v unexpectedly contain %q", keys, excluded)
		}
	}
	for _, required := range []serverAction{serverActionProjects, serverActionLogs, serverActionServices, serverActionFirewall, serverActionMetrics, serverActionReboot, serverActionSSH} {
		if !slices.Contains(keys, required) {
			t.Errorf("Docker server actions %v do not contain %q", keys, required)
		}
	}
}

func TestDockerApplicationActionsFollowLifecycleState(t *testing.T) {
	deployedAt := time.Now()
	tests := []struct {
		name     string
		app      api.DockerApplicationResponse
		want     []dockerApplicationMenuAction
		dontWant []dockerApplicationMenuAction
	}{
		{
			name:     "never deployed",
			app:      api.DockerApplicationResponse{Status: "idle"},
			want:     []dockerApplicationMenuAction{dockerApplicationDetails, dockerApplicationDeploy, dockerApplicationDeployments},
			dontWant: []dockerApplicationMenuAction{dockerApplicationReload, dockerApplicationStart, dockerApplicationStop},
		},
		{
			name:     "running",
			app:      api.DockerApplicationResponse{Status: "running", LastDeployedAt: &deployedAt},
			want:     []dockerApplicationMenuAction{dockerApplicationReload, dockerApplicationStop},
			dontWant: []dockerApplicationMenuAction{dockerApplicationStart},
		},
		{
			name:     "stopped",
			app:      api.DockerApplicationResponse{Status: "stopped", LastDeployedAt: &deployedAt},
			want:     []dockerApplicationMenuAction{dockerApplicationReload, dockerApplicationStart},
			dontWant: []dockerApplicationMenuAction{dockerApplicationStop},
		},
		{
			name:     "building",
			app:      api.DockerApplicationResponse{Status: "building", LastDeployedAt: &deployedAt},
			dontWant: []dockerApplicationMenuAction{dockerApplicationDeploy, dockerApplicationReload, dockerApplicationStart, dockerApplicationStop},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := dockerApplicationActionOptions(tt.app)
			keys := make([]dockerApplicationMenuAction, 0, len(options))
			for _, option := range options {
				keys = append(keys, option.key)
			}
			for _, wanted := range tt.want {
				if !slices.Contains(keys, wanted) {
					t.Errorf("actions %v do not contain %q", keys, wanted)
				}
			}
			for _, unwanted := range tt.dontWant {
				if slices.Contains(keys, unwanted) {
					t.Errorf("actions %v unexpectedly contain %q", keys, unwanted)
				}
			}
		})
	}
}

func TestWaitForDockerDeploymentTaskPollsUntilTaskIsAttached(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/servers/server-1/docker/projects/project-1/applications/app-1/deployments" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		call := calls.Add(1)
		task := ""
		if call > 1 {
			task = `,"task_id":"task-1"`
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"success":true,"data":[{"id":"deployment-1","status":"pending"%s}]}`, task)
	}))
	defer server.Close()

	client := api.NewClient(&config.Config{APIURL: server.URL + "/api"})
	got, err := waitForDockerDeploymentTask(
		client,
		"server-1",
		"project-1",
		"app-1",
		api.DockerDeploymentResponse{ID: "deployment-1", Status: "pending"},
		2*time.Second,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.TaskID == nil || *got.TaskID != "task-1" {
		t.Fatalf("deployment task = %v, want task-1", got.TaskID)
	}
	if calls.Load() < 2 {
		t.Fatalf("poll calls = %d, want at least 2", calls.Load())
	}
}

func TestDockerSourceSummaryRedactsRepositoryCredentials(t *testing.T) {
	summary := dockerSourceSummary(api.DockerApplicationResponse{SourceConfig: map[string]any{
		"repo":   "https://user:secret@github.com/acme/api.git?access_token=token",
		"branch": "main",
	}})
	if strings.Contains(summary, "secret") || strings.Contains(summary, "token=token") {
		t.Fatalf("source summary leaked repository credentials: %q", summary)
	}
}
