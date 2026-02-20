package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	Indigo    = lipgloss.Color("#818CF8")
	Green     = lipgloss.Color("#4ade80")
	Slate     = lipgloss.Color("#94a3b8")
	Red       = lipgloss.Color("#f87171")
	Yellow    = lipgloss.Color("#fbbf24")
	Blue      = lipgloss.Color("#60a5fa")
	Cyan      = lipgloss.Color("#22d3ee")
	Orange    = lipgloss.Color("#fb923c")
	White     = lipgloss.Color("#f8fafc")
	Muted     = lipgloss.Color("#64748b")
	DarkSlate = lipgloss.Color("#334155")

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

func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "running", "connected", "finished", "installed", "active", "healthy":
		return StatusConnected
	case "failed", "disconnected", "error", "unhealthy":
		return StatusDisconnected
	case "pending", "new", "starting", "installing", "deploying":
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
