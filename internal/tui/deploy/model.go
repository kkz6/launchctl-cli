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
}

func NewModel(siteName string, ws *api.WSClient, channel string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(tui.Green)

	vp := viewport.New(80, 20)

	return Model{
		viewport: vp,
		spinner:  s,
		status:   "pending",
		siteName: siteName,
		ws:       ws,
		channel:  channel,
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
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 4
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight
		m.viewport.SetContent(strings.Join(m.logs, "\n"))

	case LogMsg:
		m.logs = append(m.logs, msg.Line)
		m.viewport.SetContent(strings.Join(m.logs, "\n"))
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

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var b strings.Builder

	header := tui.Title.Render(fmt.Sprintf("Deploying: %s", m.siteName))
	b.WriteString(header + "\n")

	if m.done {
		statusStyle := tui.StatusStyle(m.status)
		b.WriteString(statusStyle.Render(fmt.Sprintf("Status: %s", m.status)) + "\n")
	} else {
		b.WriteString(m.spinner.View() + " " + tui.Dim.Render(m.status))
		if m.step != "" {
			b.WriteString(" - " + tui.Info.Render(m.step))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.viewport.View())

	if m.done {
		b.WriteString("\n" + tui.Dim.Render("Press q to exit"))
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
