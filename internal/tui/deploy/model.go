package deploy

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui"
)

type LogMsg struct {
	Line string
}

type StatusMsg struct {
	Status string
	Step   string
}

type DoneMsg struct {
	Status string
}

type ErrorMsg struct {
	Err error
}

var (
	logBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tui.DarkSlate).
			Padding(0, 1)

	logHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(tui.Slate).
			Background(lipgloss.Color("#1e293b")).
			Padding(0, 1).
			MarginBottom(0)

	statusBarStyle = lipgloss.NewStyle().
			Padding(0, 1).
			MarginBottom(1)

	stepStyle = lipgloss.NewStyle().
			Foreground(tui.Cyan).
			Bold(true)

	logLineStyle = lipgloss.NewStyle().
			Foreground(tui.Slate)

	logCountStyle = lipgloss.NewStyle().
			Foreground(tui.Muted)
)

type Model struct {
	viewport viewport.Model
	spinner  spinner.Model
	logs     []string
	status   string
	step     string
	done     bool
	err      error
	siteName string
	width    int
	height   int
	ws       *api.WSClient
	channel  string
	showLogs bool
}

func NewModel(siteName string, ws *api.WSClient, channel string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(tui.Indigo)

	vp := viewport.New(80, 20)

	return Model{
		viewport: vp,
		spinner:  s,
		status:   "pending",
		siteName: siteName,
		ws:       ws,
		channel:  channel,
		showLogs: true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		listenWS(m.ws),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "l":
			m.showLogs = !m.showLogs
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 8
		m.viewport.Width = msg.Width - 4
		if m.showLogs {
			m.viewport.Height = msg.Height - headerHeight
		}
		m.viewport.SetContent(m.renderLogs())

	case LogMsg:
		m.logs = append(m.logs, msg.Line)
		m.viewport.SetContent(m.renderLogs())
		m.viewport.GotoBottom()
		if !m.done {
			cmds = append(cmds, listenWS(m.ws))
		}

	case StatusMsg:
		m.status = msg.Status
		m.step = msg.Step
		if !m.done {
			cmds = append(cmds, listenWS(m.ws))
		}

	case DoneMsg:
		m.done = true
		m.status = msg.Status

	case ErrorMsg:
		m.err = msg.Err
		m.done = true

	case spinner.TickMsg:
		if !m.done {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	if m.showLogs {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) renderLogs() string {
	if len(m.logs) == 0 {
		return logLineStyle.Render("  Waiting for output...")
	}

	var b strings.Builder
	for _, line := range m.logs {
		b.WriteString(logLineStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) View() string {
	var b strings.Builder
	w := m.width
	if w == 0 {
		w = 80
	}

	// Title
	title := tui.Title.Render(fmt.Sprintf("  Deploying: %s", m.siteName))
	b.WriteString(title + "\n")

	// Status bar
	var statusLine string
	if m.done {
		statusStyle := tui.StatusStyle(m.status)
		icon := "\u2713"
		if m.status == "failed" || m.status == "timeout" || m.status == "cancelled" {
			icon = "\u2717"
		}
		statusLine = statusStyle.Render(fmt.Sprintf("  %s  %s", icon, m.status))
	} else {
		statusLine = "  " + m.spinner.View() + " " + tui.Value.Render(m.status)
		if m.step != "" {
			statusLine += "  " + stepStyle.Render(m.step)
		}
	}
	b.WriteString(statusBarStyle.Render(statusLine) + "\n")

	// Log panel
	if m.showLogs {
		logHeader := logHeaderStyle.Render(fmt.Sprintf(" Output  %s",
			logCountStyle.Render(fmt.Sprintf("(%d lines)", len(m.logs)))))
		b.WriteString("  " + logHeader + "\n")

		logBox := logBorderStyle.Width(w - 4)
		b.WriteString("  " + logBox.Render(m.viewport.View()) + "\n")
	} else {
		collapsed := tui.Dim.Render(fmt.Sprintf("  Logs hidden (%d lines) - press l to show", len(m.logs)))
		b.WriteString(collapsed + "\n")
	}

	// Footer
	b.WriteString("\n")
	if m.done {
		b.WriteString(tui.Dim.Render("  q quit  l toggle logs"))
	} else {
		b.WriteString(tui.Dim.Render("  l toggle logs  scroll \u2191/\u2193"))
	}

	return b.String()
}

func listenWS(ws *api.WSClient) tea.Cmd {
	return func() tea.Msg {
		msg, err := ws.ReadMessage()
		if err != nil {
			return ErrorMsg{Err: err}
		}

		switch msg.Event {
		case "deployment.log":
			var event api.DeploymentLogEvent
			if err := parseEventData(msg.Data, &event); err == nil {
				return LogMsg{Line: event.Output}
			}

		case "deployment.progress":
			var event api.DeploymentLogEvent
			if err := parseEventData(msg.Data, &event); err == nil {
				return StatusMsg{Status: event.Status, Step: event.Step}
			}

		case "deployment.finished":
			return DoneMsg{Status: "finished"}

		case "deployment.failed":
			return DoneMsg{Status: "failed"}

		case "deployment.timeout":
			return DoneMsg{Status: "timeout"}

		case "deployment.cancelled":
			return DoneMsg{Status: "cancelled"}

		case "deployment.started":
			var event api.DeploymentLogEvent
			if err := parseEventData(msg.Data, &event); err == nil {
				return StatusMsg{Status: "deploying", Step: event.Step}
			}
		}

		return listenWS(ws)()
	}
}

func parseEventData(data []byte, v any) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}

	return api.UnmarshalEventData(data, v)
}
