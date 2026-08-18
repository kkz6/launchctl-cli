package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	Indigo = lipgloss.AdaptiveColor{Light: "#4F46E5", Dark: "#818CF8"}
	Green  = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	Slate  = lipgloss.AdaptiveColor{Light: "#475569", Dark: "#94A3B8"}
	Red    = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
	Yellow = lipgloss.AdaptiveColor{Light: "#A16207", Dark: "#FBBF24"}
	Blue   = lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#60A5FA"}
	Cyan   = lipgloss.AdaptiveColor{Light: "#0E7490", Dark: "#22D3EE"}
	Orange = lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#FB923C"}

	// White is the historical name for primary text. Its light variant is
	// intentionally dark so values remain readable on light terminal themes.
	White     = lipgloss.AdaptiveColor{Light: "#0F172A", Dark: "#F8FAFC"}
	Muted     = lipgloss.AdaptiveColor{Light: "#64748B", Dark: "#94A3B8"}
	DarkSlate = lipgloss.AdaptiveColor{Light: "#64748B", Dark: "#64748B"}
	Panel     = lipgloss.AdaptiveColor{Light: "#E2E8F0", Dark: "#1E293B"}
	OnAccent  = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#0F172A"}

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Indigo)

	Subtitle = lipgloss.NewStyle().
			Foreground(Slate)

	Success = lipgloss.NewStyle().
		Foreground(Green).
		Bold(true)

	Error = lipgloss.NewStyle().
		Foreground(Red).
		Bold(true)

	Warning = lipgloss.NewStyle().
		Foreground(Yellow).
		Bold(true)

	Info = lipgloss.NewStyle().
		Foreground(Blue)

	Dim = lipgloss.NewStyle().
		Foreground(Muted)

	Bold = lipgloss.NewStyle().
		Bold(true).
		Foreground(White)

	Label = lipgloss.NewStyle().
		Foreground(Slate).
		Width(18)

	Value = lipgloss.NewStyle().
		Foreground(White)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Indigo).
			Padding(1, 2)

	StatusConnected    = lipgloss.NewStyle().Foreground(Green).Bold(true)
	StatusDisconnected = lipgloss.NewStyle().Foreground(Red).Bold(true)
	StatusPending      = lipgloss.NewStyle().Foreground(Yellow).Bold(true)
	StatusRunning      = lipgloss.NewStyle().Foreground(Blue).Bold(true)
)

// ConfigureTheme applies an optional terminal-background override. Automatic
// detection remains the default, while the override keeps light themes usable
// in multiplexers that cannot report their background color.
func ConfigureTheme(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return nil
	case "light":
		lipgloss.SetHasDarkBackground(false)
		return nil
	case "dark":
		lipgloss.SetHasDarkBackground(true)
		return nil
	default:
		return fmt.Errorf("LAUNCHCTL_THEME must be auto, light, or dark")
	}
}

func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "running", "success", "connected", "finished", "installed", "active", "healthy":
		return StatusConnected
	case "failed", "errored", "disconnected", "error", "unhealthy":
		return StatusDisconnected
	case "pending", "new", "starting", "stopping", "restarting", "building", "installing", "deploying", "deleting":
		return StatusPending
	case "provisioning", "rebooting":
		return StatusRunning
	default:
		return Dim
	}
}

// ShowSuccess displays a success message
func ShowSuccess(message string) {
	fmt.Println(Success.Render("\u2713 " + message))
}

// ShowError displays an error message
func ShowError(message string) {
	fmt.Println(Error.Render("\u2717 " + message))
}

// ShowWarning displays a warning message
func ShowWarning(message string) {
	fmt.Println(Warning.Render("\u26a0 " + message))
}

// ShowInfo displays an info message
func ShowInfo(message string) {
	fmt.Println(Info.Render("\u25b8 " + message))
}

// CreateBox creates a styled box around content
func CreateBox(title, content string, width int) string {
	box := BoxStyle.Width(width)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(Indigo).
		MarginBottom(1)

	fullContent := titleStyle.Render(title) + "\n" + content
	return box.Render(fullContent)
}
