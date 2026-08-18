package terminal

import (
	"net/url"
	"testing"

	"github.com/kkz6/launchctl/internal/config"
)

func TestBuildURLUsesAPIRootWebSocketPath(t *testing.T) {
	got, err := buildURL(&config.Config{APIURL: "https://api.launchctl.io"}, Options{
		ServerID: "server-a",
		SiteID:   "site-a",
		Username: "launcher",
		Token:    "secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "wss" || parsed.Host != "api.launchctl.io" || parsed.Path != "/terminal/ws" {
		t.Fatalf("unexpected terminal URL: %s", got)
	}
	if parsed.Query().Get("serverId") != "server-a" || parsed.Query().Get("siteId") != "site-a" {
		t.Fatalf("terminal URL is missing resource identifiers: %s", got)
	}
}

func TestBuildURLPreservesLegacyAPIBasePath(t *testing.T) {
	got, err := buildURL(&config.Config{APIURL: "https://staging.example/api"}, Options{
		ServerID: "server-a",
		Token:    "secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/api/terminal/ws" {
		t.Fatalf("terminal path = %q, want /api/terminal/ws", parsed.Path)
	}
}
