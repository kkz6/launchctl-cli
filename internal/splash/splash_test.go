package splash

import (
	"strings"
	"testing"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderPlainWide(t *testing.T) {
	want := "launchctl  v0.2.2\n" + Description + "\n"
	got := Render("0.2.2", Options{Width: 80})

	if got != want {
		t.Fatalf("unexpected splash:\nwant %q\n got %q", want, got)
	}
}

func TestRenderWrapsForNarrowPanes(t *testing.T) {
	want := "launchctl  v0.2.2\nDeploy, operate, and monitor\nfrom one terminal.\n"
	got := Render("v0.2.2", Options{Width: 28})

	if got != want {
		t.Fatalf("unexpected narrow splash:\nwant %q\n got %q", want, got)
	}
}

func TestRenderNeverExceedsRequestedWidth(t *testing.T) {
	for _, width := range []int{1, 8, 9, 16, 28, 80} {
		got := Render("123.456.789-development", Options{
			Width:         width,
			Color:         true,
			UpdateVersion: "987.654.321-development",
		})
		for lineNumber, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
			if gotWidth := ansi.StringWidth(line); gotWidth > width {
				t.Fatalf("width %d, line %d is %d cells: %q", width, lineNumber+1, gotWidth, line)
			}
		}
	}
}

func TestRenderIncludesUpdateNotice(t *testing.T) {
	want := "launchctl  v0.2.2\n" + Description + "\n" +
		"Update available: v0.3.0 · run lctl update\n"
	got := Render("0.2.2", Options{Width: 80, UpdateVersion: "0.3.0"})

	if got != want {
		t.Fatalf("unexpected splash with update notice:\nwant %q\n got %q", want, got)
	}
}

func TestRenderUsesCompactUpdateNoticeInNarrowPanes(t *testing.T) {
	want := "launchctl\nv0.2.2\nUpdate: v0.3.0\n"
	got := Render("0.2.2", Options{Width: 16, UpdateVersion: "v0.3.0"})

	if got != want {
		t.Fatalf("unexpected narrow update notice:\nwant %q\n got %q", want, got)
	}
}

func TestRenderSuppressesMissingOrDevelopmentUpdateNotice(t *testing.T) {
	want := Render("0.2.2", Options{Width: 80})
	for _, updateVersion := range []string{"", " ", "dev"} {
		t.Run(updateVersion, func(t *testing.T) {
			got := Render("0.2.2", Options{Width: 80, UpdateVersion: updateVersion})
			if got != want {
				t.Fatalf("update version %q changed the splash:\nwant %q\n got %q", updateVersion, want, got)
			}
		})
	}
}

func TestRenderUsesCompactBrandInVeryNarrowPanes(t *testing.T) {
	want := "lctl\nv0.2.2\n"
	got := Render("0.2.2", Options{Width: 8})

	if got != want {
		t.Fatalf("unexpected compact splash:\nwant %q\n got %q", want, got)
	}
}

func TestRenderColorIsOptionalAndSemantic(t *testing.T) {
	plain := Render("0.2.2", Options{Width: 80})
	colored := Render("0.2.2", Options{Width: 80, Color: true})

	if strings.Contains(plain, "\x1b[") {
		t.Fatalf("plain output contains ANSI escapes: %q", plain)
	}
	for _, r := range plain {
		if unicode.IsControl(r) && r != '\n' {
			t.Fatalf("plain output contains control character %U", r)
		}
	}
	if !strings.Contains(colored, "\x1b[") {
		t.Fatalf("colored output does not contain ANSI styling: %q", colored)
	}
	if strings.Contains(colored, "\x1b]") || strings.Contains(colored, "\x1b[6n") {
		t.Fatalf("colored output contains a terminal query: %q", colored)
	}
	if got := ansi.Strip(colored); got != plain {
		t.Fatalf("color changed semantic output:\nwant %q\n got %q", plain, got)
	}
}

func TestDisplayVersion(t *testing.T) {
	tests := map[string]string{
		"":       "dev",
		"  ":     "dev",
		"0.2.2":  "v0.2.2",
		"v0.2.2": "v0.2.2",
	}

	for input, want := range tests {
		if got := displayVersion(input); got != want {
			t.Errorf("displayVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestShouldRenderPolicy(t *testing.T) {
	tests := []struct {
		name       string
		isTTY      bool
		ci         bool
		json       bool
		term       string
		wantRender bool
	}{
		{name: "interactive", isTTY: true, term: "xterm-256color", wantRender: true},
		{name: "redirected", term: "xterm-256color"},
		{name: "ci", isTTY: true, ci: true, term: "xterm-256color"},
		{name: "json", isTTY: true, json: true, term: "xterm-256color"},
		{name: "dumb terminal", isTTY: true, term: "dumb"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldRender(test.isTTY, test.ci, test.json, test.term); got != test.wantRender {
				t.Fatalf("shouldRender() = %t, want %t", got, test.wantRender)
			}
		})
	}
}

func TestInteractivePolicyRequiresInputAndOutputTTYs(t *testing.T) {
	tests := []struct {
		name      string
		inputTTY  bool
		outputTTY bool
		ci        bool
		json      bool
		term      string
		want      bool
	}{
		{name: "interactive", inputTTY: true, outputTTY: true, term: "screen-256color", want: true},
		{name: "redirected input", outputTTY: true, term: "screen-256color"},
		{name: "redirected output", inputTTY: true, term: "screen-256color"},
		{name: "ci", inputTTY: true, outputTTY: true, ci: true, term: "screen-256color"},
		{name: "json", inputTTY: true, outputTTY: true, json: true, term: "screen-256color"},
		{name: "dumb terminal", inputTTY: true, outputTTY: true, term: "dumb"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := isInteractive(test.inputTTY, test.outputTTY, test.ci, test.json, test.term)
			if got != test.want {
				t.Fatalf("isInteractive() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestTerminalColorPolicy(t *testing.T) {
	tests := []struct {
		name     string
		isTTY    bool
		term     string
		noColor  bool
		cliColor string
		want     bool
	}{
		{name: "interactive", isTTY: true, term: "xterm-256color", want: true},
		{name: "redirected", term: "xterm-256color"},
		{name: "no color", isTTY: true, term: "xterm-256color", noColor: true},
		{name: "cli color disabled", isTTY: true, term: "xterm-256color", cliColor: "0"},
		{name: "dumb terminal", isTTY: true, term: "dumb"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := terminalOptions(test.isTTY, 72, test.term, test.noColor, test.cliColor)
			if got.Width != 72 || got.Color != test.want {
				t.Fatalf("terminalOptions() = %#v, want width 72 and color %t", got, test.want)
			}
		})
	}
}
