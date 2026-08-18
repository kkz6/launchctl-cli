package servers

import (
	"reflect"
	"testing"

	"github.com/kkz6/launchctl/internal/api"
)

func TestServerResourceCountsFollowServerType(t *testing.T) {
	tests := []struct {
		name        string
		server      api.ServerResponse
		wantCounts  []serverResourceCount
		wantSummary string
	}{
		{
			name: "Docker",
			server: api.ServerResponse{
				Type:           "Docker",
				SitesCount:     7,
				ProjectsCount:  2,
				WorkloadsCount: 6,
			},
			wantCounts: []serverResourceCount{
				{Label: "Projects", Count: 2},
				{Label: "Workloads", Count: 6},
			},
			wantSummary: "2 projects / 6 workloads",
		},
		{
			name: "PHP",
			server: api.ServerResponse{
				Type:          "php",
				SitesCount:    1,
				ProjectsCount: 9,
			},
			wantCounts:  []serverResourceCount{{Label: "Sites", Count: 1}},
			wantSummary: "1 site",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serverResourceCounts(tt.server); !reflect.DeepEqual(got, tt.wantCounts) {
				t.Fatalf("serverResourceCounts() = %#v, want %#v", got, tt.wantCounts)
			}
			if got := serverResourceSummary(tt.server); got != tt.wantSummary {
				t.Fatalf("serverResourceSummary() = %q, want %q", got, tt.wantSummary)
			}
		})
	}
}
