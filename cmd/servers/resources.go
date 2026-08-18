package servers

import (
	"fmt"
	"strings"

	"github.com/kkz6/launchctl/internal/api"
)

type serverResourceCount struct {
	Label string
	Count int
}

func serverResourceCounts(server api.ServerResponse) []serverResourceCount {
	if strings.EqualFold(server.Type, "docker") {
		return []serverResourceCount{
			{Label: "Projects", Count: server.ProjectsCount},
			{Label: "Workloads", Count: server.WorkloadsCount},
		}
	}
	return []serverResourceCount{{Label: "Sites", Count: server.SitesCount}}
}

func serverResourceSummary(server api.ServerResponse) string {
	counts := serverResourceCounts(server)
	parts := make([]string, 0, len(counts))
	for _, resource := range counts {
		parts = append(parts, fmt.Sprintf("%d %s", resource.Count, resourceNoun(resource.Label, resource.Count)))
	}
	return strings.Join(parts, " / ")
}

func resourceNoun(label string, count int) string {
	noun := strings.ToLower(label)
	if count == 1 {
		noun = strings.TrimSuffix(noun, "s")
	}
	return noun
}
