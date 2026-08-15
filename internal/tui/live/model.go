package live

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui"
)

const maxLines = 5000

type Options struct {
	Title        string
	Subtitle     string
	InitialLines []string
	Filter       func(*api.WSMessage) bool
	WS           *api.WSClient
}

type Model struct {
	viewport viewport.Model
	ws       *api.WSClient
	title    string
	subtitle string
	filter   func(*api.WSMessage) bool
	lines    []string
	state    api.WSState
	paused   bool
	dropped  int
	width    int
	height   int
}

type eventMsg struct{ message *api.WSMessage }
type stateMsg struct{ state api.WSState }
type readErrorMsg struct{ err error }

func NewModel(options Options) Model {
	view := viewport.New(90, 20)
	lines := append([]string(nil), options.InitialLines...)
	view.SetContent(strings.Join(lines, "\n"))
	return Model{
		viewport: view,
		ws:       options.WS,
		title:    options.Title,
		subtitle: options.Subtitle,
		filter:   options.Filter,
		lines:    lines,
		state:    api.WSState{State: api.StateReconnecting},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(waitEvent(m.ws), waitState(m.ws))
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		switch message.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case " ", "p":
			m.paused = !m.paused
		case "c":
			m.lines = nil
			m.dropped = 0
			m.viewport.SetContent("")
		}
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
		m.viewport.Width = max(20, message.Width-4)
		m.viewport.Height = max(3, message.Height-8)
		m.viewport.SetContent(strings.Join(m.lines, "\n"))
	case eventMsg:
		if m.filter == nil || m.filter(message.message) {
			if m.paused {
				m.dropped++
			} else {
				m.lines = append(m.lines, formatEvent(message.message, time.Now()))
				if len(m.lines) > maxLines {
					m.lines = append([]string(nil), m.lines[len(m.lines)-maxLines:]...)
				}
				m.viewport.SetContent(strings.Join(m.lines, "\n"))
				m.viewport.GotoBottom()
			}
		}
		return m, waitEvent(m.ws)
	case stateMsg:
		m.state = message.state
		return m, waitState(m.ws)
	case readErrorMsg:
		m.state = api.WSState{State: api.StateClosed, Err: message.err}
	}

	var command tea.Cmd
	m.viewport, command = m.viewport.Update(message)
	return m, command
}

func (m Model) View() string {
	width := m.width
	if width == 0 {
		width = 94
	}
	stateText := string(m.state.State)
	stateColor := tui.Green
	if m.state.State == api.StateReconnecting {
		stateText = fmt.Sprintf("reconnecting · attempt %d", m.state.Attempt)
		stateColor = tui.Yellow
	} else if m.state.State == api.StateClosed || m.state.State == api.StateDisconnected {
		stateColor = tui.Red
	}

	var builder strings.Builder
	builder.WriteString(tui.Title.Render(m.title))
	builder.WriteString("  ")
	builder.WriteString(lipgloss.NewStyle().Foreground(stateColor).Render("● " + stateText))
	builder.WriteString("\n")
	if m.subtitle != "" {
		builder.WriteString(tui.Dim.Render(m.subtitle))
		builder.WriteString("\n")
	}
	if m.paused {
		builder.WriteString(tui.Warning.Render(fmt.Sprintf("PAUSED · %d events skipped", m.dropped)))
	} else {
		builder.WriteString(tui.Dim.Render(fmt.Sprintf("%d lines · newest events follow", len(m.lines))))
	}
	builder.WriteString("\n\n")
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(tui.DarkSlate).Padding(0, 1).Width(max(20, width-4))
	builder.WriteString(box.Render(m.viewport.View()))
	builder.WriteString("\n\n")
	builder.WriteString(tui.Dim.Render("q quit   space pause   c clear   ↑/↓ scroll   pgup/pgdn"))
	return builder.String()
}

func waitEvent(ws *api.WSClient) tea.Cmd {
	return func() tea.Msg {
		message, err := ws.ReadMessage()
		if err != nil {
			return readErrorMsg{err: err}
		}
		return eventMsg{message: message}
	}
}

func waitState(ws *api.WSClient) tea.Cmd {
	return func() tea.Msg {
		state, ok := <-ws.States()
		if !ok {
			return stateMsg{state: api.WSState{State: api.StateClosed}}
		}
		return stateMsg{state: state}
	}
}

func formatEvent(message *api.WSMessage, now time.Time) string {
	timestamp := lipgloss.NewStyle().Foreground(tui.Muted).Render(now.Format("15:04:05"))
	event := lipgloss.NewStyle().Foreground(tui.Cyan).Bold(true).Render(message.Event)
	channel := lipgloss.NewStyle().Foreground(tui.Indigo).Render(message.Channel)
	detail := eventDetail(message.Data)
	if detail == "" {
		return fmt.Sprintf("%s  %-28s  %s", timestamp, event, channel)
	}
	return fmt.Sprintf("%s  %-28s  %s\n          %s", timestamp, event, channel, detail)
}

func eventDetail(data json.RawMessage) string {
	if len(data) == 0 || string(data) == "null" {
		return ""
	}
	var fields map[string]any
	if json.Unmarshal(data, &fields) == nil {
		for _, key := range []string{"output", "message", "error", "step", "status"} {
			if value, ok := fields[key]; ok && fmt.Sprint(value) != "" {
				return strings.TrimSpace(fmt.Sprint(value))
			}
		}
	}
	var compact bytes.Buffer
	if json.Compact(&compact, data) == nil {
		return compact.String()
	}
	return strings.TrimSpace(string(data))
}
