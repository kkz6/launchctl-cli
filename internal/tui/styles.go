package tui

import "github.com/charmbracelet/lipgloss"

var (
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
		Foreground(Green)

	Subtitle = lipgloss.NewStyle().
			Foreground(Slate)

	Success = lipgloss.NewStyle().
		Foreground(Green)

	Error = lipgloss.NewStyle().
		Foreground(Red)

	Warning = lipgloss.NewStyle().
		Foreground(Yellow)

	Info = lipgloss.NewStyle().
		Foreground(Blue)

	Dim = lipgloss.NewStyle().
		Foreground(Muted)

	Bold = lipgloss.NewStyle().
		Bold(true).
		Foreground(White)

	Label = lipgloss.NewStyle().
		Foreground(Slate).
		Width(20)

	Value = lipgloss.NewStyle().
		Foreground(White)

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
