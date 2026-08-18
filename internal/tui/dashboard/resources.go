package dashboard

import (
	"fmt"
	"strings"

	"github.com/kkz6/launchctl/internal/api"
)

func dashboardResourceSummary(server api.DashboardServer) string {
	if strings.EqualFold(server.Type, "docker") {
		return dashboardCountLabel(server.WorkloadsCount, "workload", "workloads")
	}
	return dashboardCountLabel(server.SitesCount, "site", "sites")
}

func dashboardCountLabel(count int, singular, plural string) string {
	label := plural
	if count == 1 {
		label = singular
	}
	return fmt.Sprintf("%d %s", count, label)
}
