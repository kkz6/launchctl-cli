package dashboard

import (
	"strings"
	"testing"

	"github.com/kkz6/launchctl/internal/api"
)

func TestDashboardResourceSummaryFollowsServerType(t *testing.T) {
	tests := []struct {
		name   string
		server api.DashboardServer
		want   string
	}{
		{
			name: "Docker uses workloads",
			server: api.DashboardServer{
				Type:           "Docker",
				SitesCount:     4,
				WorkloadsCount: 6,
			},
			want: "6 workloads",
		},
		{
			name: "PHP uses sites",
			server: api.DashboardServer{
				Type:           "php",
				SitesCount:     1,
				WorkloadsCount: 9,
			},
			want: "1 site",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dashboardResourceSummary(tt.server); got != tt.want {
				t.Fatalf("dashboardResourceSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderServersLabelsMixedResources(t *testing.T) {
	model := Model{data: &api.DashboardResponse{Servers: []api.DashboardServer{
		{Name: "Docker", Type: "docker", WorkloadsCount: 6},
		{Name: "PHP", Type: "php", SitesCount: 1},
	}}}

	view := model.renderServers()
	for _, want := range []string{"Resources", "6 workloads", "1 site"} {
		if !strings.Contains(view, want) {
			t.Fatalf("server table is missing %q:\n%s", want, view)
		}
	}
}
