package splash

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/termenv"
)

const (
	// Description is deliberately short so the brand header remains useful in
	// narrow terminal and tmux panes.
	Description             = "Deploy, operate, and monitor from one terminal."
	defaultWidth            = 80
	minimumDescriptionWidth = 28
	// InteractiveDisplayDuration keeps the standalone startup splash visible
	// before the authenticated navigation menu replaces it.
	InteractiveDisplayDuration = 2 * time.Second
)

// Options controls the deterministic parts of the splash renderer. Keeping
// terminal detection outside Render makes the output easy to preview and test.
type Options struct {
	Width         int
	Color         bool
	UpdateVersion string
}

// Render returns a compact, persistent product header. It never probes the
// terminal background and never sleeps or writes to the terminal directly.
func Render(version string, options Options) string {
	width := options.Width
	if width < 1 {
		width = defaultWidth
	}

	renderer := lipgloss.NewRenderer(io.Discard)
	profile := termenv.Ascii
	if options.Color {
		// Basic ANSI colors follow the user's terminal theme and work in tmux.
		profile = termenv.ANSI
	}
	renderer.SetColorProfile(profile)

	brandStyle := renderer.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("2"))
	metaStyle := renderer.NewStyle().Foreground(lipgloss.Color("8"))
	updateStyle := renderer.NewStyle().Foreground(lipgloss.Color("3"))

	product := "launchctl"
	if ansi.StringWidth(product) > width {
		product = "lctl"
	}
	version = displayVersion(version)

	var lines []string
	if ansi.StringWidth(product)+2+ansi.StringWidth(version) <= width {
		lines = append(lines, brandStyle.Render(product)+"  "+metaStyle.Render(version))
	} else {
		lines = append(lines, brandStyle.Render(ansi.Truncate(product, width, "")))
		lines = append(lines, metaStyle.Render(ansi.Truncate(version, width, "")))
	}

	if width >= minimumDescriptionWidth {
		for _, line := range wrappedLines(Description, width) {
			lines = append(lines, metaStyle.Render(line))
		}
	}
	if updateVersion := displayUpdateVersion(options.UpdateVersion); updateVersion != "" {
		notice := "Update available: " + updateVersion + " · run lctl update"
		if width < 24 {
			notice = "Update: " + updateVersion
		}
		for _, line := range wrappedLines(notice, width) {
			lines = append(lines, updateStyle.Render(line))
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func wrappedLines(value string, width int) []string {
	var lines []string
	current := ""
	flush := func() {
		if current != "" {
			lines = append(lines, current)
			current = ""
		}
	}

	for _, word := range strings.Fields(value) {
		chunks := strings.Split(ansi.Hardwrap(word, width, false), "\n")
		for index, chunk := range chunks {
			if current == "" {
				current = chunk
			} else if ansi.StringWidth(current)+1+ansi.StringWidth(chunk) <= width {
				current += " " + chunk
			} else {
				flush()
				current = chunk
			}

			// Every non-final chunk filled a line while breaking an oversized
			// word, so the next chunk must begin on a fresh line.
			if index < len(chunks)-1 {
				flush()
			}
		}
	}
	flush()
	return lines
}

// ShouldRender reports whether decorative root output is appropriate. Bare
// commands in scripts and CI remain plain and non-interactive.
func ShouldRender(out *os.File, ciMode, jsonOutput bool) bool {
	isTTY := out != nil && term.IsTerminal(out.Fd())
	return shouldRender(isTTY, ciMode, jsonOutput, os.Getenv("TERM"))
}

// IsInteractive reports whether both sides of an interactive TUI are attached
// to a terminal. This prevents a bare command with redirected input from
// opening navigation and waiting for keys that can never arrive.
func IsInteractive(in, out *os.File, ciMode, jsonOutput bool) bool {
	inputTTY := in != nil && term.IsTerminal(in.Fd())
	outputTTY := out != nil && term.IsTerminal(out.Fd())
	return isInteractive(inputTTY, outputTTY, ciMode, jsonOutput, os.Getenv("TERM"))
}

// TerminalOptions detects only terminal capabilities that are available
// immediately. In particular, it does not issue an OSC background query.
func TerminalOptions(out *os.File) Options {
	isTTY := out != nil && term.IsTerminal(out.Fd())
	width := defaultWidth
	if isTTY {
		if detected, _, err := term.GetSize(out.Fd()); err == nil && detected > 0 {
			width = detected
		}
	}

	_, noColor := os.LookupEnv("NO_COLOR")
	return terminalOptions(isTTY, width, os.Getenv("TERM"), noColor, os.Getenv("CLICOLOR"))
}

func displayVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "dev"
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func displayUpdateVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || version == "dev" {
		return ""
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func shouldRender(isTTY, ciMode, jsonOutput bool, termName string) bool {
	return isTTY && !ciMode && !jsonOutput && !strings.EqualFold(termName, "dumb")
}

func isInteractive(inputTTY, outputTTY, ciMode, jsonOutput bool, termName string) bool {
	return inputTTY && shouldRender(outputTTY, ciMode, jsonOutput, termName)
}

func terminalOptions(isTTY bool, width int, termName string, noColor bool, cliColor string) Options {
	if width < 1 {
		width = defaultWidth
	}

	color := isTTY &&
		!noColor &&
		cliColor != "0" &&
		!strings.EqualFold(termName, "dumb")

	return Options{Width: width, Color: color}
}
