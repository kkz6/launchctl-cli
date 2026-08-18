package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kkz6/launchctl/internal/config"
)

func TestListServersDecodesDockerResourceCounts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/servers/" || r.URL.RawQuery != "per_page=100" {
			t.Fatalf("unexpected request %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": [{
				"id": "server-1",
				"type": "docker",
				"sites_count": null,
				"projects_count": 2,
				"workloads_count": 6
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(&config.Config{APIURL: server.URL + "/api"})
	servers, _, err := client.ListServers()
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 {
		t.Fatalf("len(servers) = %d, want 1", len(servers))
	}
	if got := servers[0].ProjectsCount; got != 2 {
		t.Fatalf("projects count = %d, want 2", got)
	}
	if got := servers[0].WorkloadsCount; got != 6 {
		t.Fatalf("workloads count = %d, want 6", got)
	}
	if got := servers[0].SitesCount; got != 0 {
		t.Fatalf("sites count = %d, want 0", got)
	}
}

func TestGetDashboardDecodesDockerWorkloadCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dashboard" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": {
				"servers": [{
					"id": "server-1",
					"type": "docker",
					"sites_count": 0,
					"workloads_count": 6
				}],
				"recent_activity": []
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(&config.Config{APIURL: server.URL + "/api"})
	dashboard, err := client.GetDashboard()
	if err != nil {
		t.Fatal(err)
	}
	if len(dashboard.Servers) != 1 {
		t.Fatalf("len(dashboard.Servers) = %d, want 1", len(dashboard.Servers))
	}
	if got := dashboard.Servers[0].Type; got != "docker" {
		t.Fatalf("server type = %q, want docker", got)
	}
	if got := dashboard.Servers[0].WorkloadsCount; got != 6 {
		t.Fatalf("workloads count = %d, want 6", got)
	}
}
