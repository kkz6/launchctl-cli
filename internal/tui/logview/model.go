package logview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui"
)

type logMsg struct {
	line string
}

type doneMsg struct{}

type errMsg struct {
	err error
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
			Padding(0, 1)

	logLineStyle = lipgloss.NewStyle().
			Foreground(tui.Slate)

	logCountStyle = lipgloss.NewStyle().
			Foreground(tui.Muted)
)

type Info struct {
	Title  string
	Status string
	Commit string
	Lines  []struct{ Label, Value string }
}

type Model struct {
	viewport viewport.Model
	spinner  spinner.Model
	info     Info
	ws       *api.LogsWSClient
	logs     []string
	done     bool
	err      error
	width    int
	height   int
}

func NewModel(info Info, ws *api.LogsWSClient) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(tui.Indigo)

	vp := viewport.New(80, 20)

	return Model{
		viewport: vp,
		spinner:  s,
		info:     info,
		ws:       ws,
	}
}

func Run(info Info, ws *api.LogsWSClient) error {
	model := NewModel(info, ws)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		listenLogs(m.ws),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := len(m.info.Lines) + 6
		m.viewport.Width = msg.Width - 2
		m.viewport.Height = msg.Height - headerHeight
		if m.viewport.Height < 3 {
			m.viewport.Height = 3
		}
		m.viewport.SetContent(m.renderLogs())

	case logMsg:
		m.logs = append(m.logs, msg.line)
		m.viewport.SetContent(m.renderLogs())
		if !m.done {
			m.viewport.GotoBottom()
			cmds = append(cmds, listenLogs(m.ws))
		}

	case doneMsg:
		m.done = true
		m.viewport.SetContent(m.renderLogs())
		m.viewport.GotoBottom()

	case errMsg:
		m.err = msg.err
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

func (m Model) renderLogs() string {
	if len(m.logs) == 0 {
		if m.done {
			return logLineStyle.Render("No output recorded for this deployment")
		}
		return logLineStyle.Render("Loading logs...")
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

	title := tui.Title.Render(m.info.Title)
	if m.info.Status != "" {
		statusStyle := tui.StatusStyle(m.info.Status)
		gap := w - lipgloss.Width(title) - lipgloss.Width(m.info.Status) - 5
		if gap < 2 {
			gap = 2
		}
		title += strings.Repeat(" ", gap) + statusStyle.Render("\u25cf "+m.info.Status)
	}
	b.WriteString(title + "\n")

	for _, l := range m.info.Lines {
		label := lipgloss.NewStyle().Foreground(tui.Slate).Width(14).Render(l.Label)
		value := lipgloss.NewStyle().Foreground(tui.White).Render(l.Value)
		b.WriteString(label + value + "\n")
	}
	b.WriteString("\n")

	logHeader := logHeaderStyle.Render(fmt.Sprintf("Output  %s",
		logCountStyle.Render(fmt.Sprintf("(%d lines)", len(m.logs)))))

	if !m.done && m.err == nil {
		logHeader += "  " + m.spinner.View()
	}
	b.WriteString(logHeader + "\n")

	logBox := logBorderStyle.Width(w - 2)
	b.WriteString(logBox.Render(m.viewport.View()) + "\n")

	if m.err != nil {
		b.WriteString(tui.Error.Render(fmt.Sprintf("\u2717 %s", m.err)) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(tui.Dim.Render("q quit  \u2191/\u2193 scroll  g/G top/bottom"))

	return b.String()
}

func listenLogs(ws *api.LogsWSClient) tea.Cmd {
	return func() tea.Msg {
		line, err := ws.ReadMessage()
		if err != nil {
			return doneMsg{}
		}
		return logMsg{line: line}
	}
}
