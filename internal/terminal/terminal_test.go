package terminal

import (
	"net/url"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/muesli/termenv"
)

func TestFrameUsesAdaptiveTerminalPalette(t *testing.T) {
	previousProfile := lipgloss.ColorProfile()
	previousBackground := lipgloss.HasDarkBackground()
	t.Cleanup(func() {
		lipgloss.SetColorProfile(previousProfile)
		lipgloss.SetHasDarkBackground(previousBackground)
	})
	lipgloss.SetColorProfile(termenv.TrueColor)

	frame := newFrame("app-server", "203.0.113.1", "launcher")
	for _, test := range []struct {
		name       string
		dark       bool
		wantSlate  string
		wantIndigo string
		wantGreen  string
	}{
		{
			name:       "light",
			wantSlate:  "38;2;71;85;105m",
			wantIndigo: "38;2;79;70;229m",
			wantGreen:  "38;2;21;128;60m",
		},
		{
			name:       "dark",
			dark:       true,
			wantSlate:  "38;2;147;163;184m",
			wantIndigo: "38;2;129;140;248m",
			wantGreen:  "38;2;73;222;128m",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			lipgloss.SetHasDarkBackground(test.dark)
			breadcrumb := frame.renderBreadcrumb()
			status := frame.renderStatus()

			if !strings.Contains(breadcrumb, test.wantSlate) || !strings.Contains(breadcrumb, test.wantIndigo) {
				t.Fatalf("breadcrumb did not use %s adaptive colors: %q", test.name, breadcrumb)
			}
			if !strings.Contains(status, test.wantGreen) {
				t.Fatalf("status did not use %s adaptive success color: %q", test.name, status)
			}
			if got, want := ansi.Strip(breadcrumb), "lctl > Servers > app-server > Terminal"; got != want {
				t.Fatalf("breadcrumb text = %q, want %q", got, want)
			}
		})
	}
}

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
