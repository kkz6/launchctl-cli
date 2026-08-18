package notify

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type Level int

const (
	LevelSuccess Level = iota
	LevelInfo
	LevelWarning
	LevelError
)

type notification struct {
	message string
	level   Level
	time    time.Time
}

var (
	mu    sync.Mutex
	queue []notification

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#60A5FA"})

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A16207", Dark: "#FBBF24"}).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}).
			Bold(true)

	barStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#64748B", Dark: "#94A3B8"})
)

func Send(title, message string) {
	Push(message, LevelInfo)
}

func Push(message string, level Level) {
	mu.Lock()
	defer mu.Unlock()
	queue = append(queue, notification{
		message: message,
		level:   level,
		time:    time.Now(),
	})
}

func Success(message string) {
	Push(message, LevelSuccess)
}

func Info(message string) {
	Push(message, LevelInfo)
}

func Warn(message string) {
	Push(message, LevelWarning)
}

func Error(message string) {
	Push(message, LevelError)
}

func Render() string {
	mu.Lock()
	defer mu.Unlock()

	if len(queue) == 0 {
		return ""
	}

	// Take the most recent notification
	n := queue[len(queue)-1]
	queue = nil

	// Only show notifications from the last 30 seconds
	if time.Since(n.time) > 30*time.Second {
		return ""
	}

	var icon, styled string
	switch n.level {
	case LevelSuccess:
		icon = "\u2713"
		styled = successStyle.Render(fmt.Sprintf("%s %s", icon, n.message))
	case LevelInfo:
		icon = "\u25b8"
		styled = infoStyle.Render(fmt.Sprintf("%s %s", icon, n.message))
	case LevelWarning:
		icon = "\u26a0"
		styled = warningStyle.Render(fmt.Sprintf("%s %s", icon, n.message))
	case LevelError:
		icon = "\u2717"
		styled = errorStyle.Render(fmt.Sprintf("%s %s", icon, n.message))
	}

	return barStyle.Render("─") + " " + styled
}
