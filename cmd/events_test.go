package cmd

import (
	"testing"

	launchapi "github.com/kkz6/launchctl/internal/api"
)

func TestBuildEventFilter(t *testing.T) {
	filter := buildEventFilter([]string{"deployment.*", "task.failed,server.updated"})
	for event, want := range map[string]bool{
		"deployment.started": true,
		"deployment.failed":  true,
		"task.failed":        true,
		"server.updated":     true,
		"site.updated":       false,
	} {
		if got := filter(&launchapi.WSMessage{Event: event}); got != want {
			t.Fatalf("filter(%q) = %v, want %v", event, got, want)
		}
	}
}
