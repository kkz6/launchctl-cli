package live

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/api"
)

func TestEventDetailPrefersHumanReadableFields(t *testing.T) {
	if got := eventDetail(json.RawMessage(`{"output":"installing nginx","status":"running"}`)); got != "installing nginx" {
		t.Fatalf("detail = %q", got)
	}
	if got := eventDetail(json.RawMessage(`{"server_id":"srv-a"}`)); got != `{"server_id":"srv-a"}` {
		t.Fatalf("compact detail = %q", got)
	}
}

func TestFormatEventIncludesRoutingContext(t *testing.T) {
	line := formatEvent(&api.WSMessage{
		Event:   "server.updated",
		Channel: "server.srv-a",
		Data:    json.RawMessage(`{"status":"active"}`),
	}, time.Date(2026, 1, 1, 12, 34, 56, 0, time.UTC))
	for _, part := range []string{"12:34:56", "server.updated", "server.srv-a", "active"} {
		if !strings.Contains(line, part) {
			t.Fatalf("formatted event %q missing %q", line, part)
		}
	}
}
