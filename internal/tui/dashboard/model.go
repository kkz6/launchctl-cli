package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui"
)

type refreshMsg struct {
	data *api.DashboardResponse
	err  error
}

type tickMsg struct{}
type liveEventMsg struct{}
type liveStateMsg struct{ state api.WSState }

type Model struct {
	client   *api.Client
	teamName string
	data     *api.DashboardResponse
	spinner  spinner.Model
	loading  bool
	err      error
	width    int
	height   int
	ws       *api.WSClient
	wsState  api.WSState
}

func NewModel(client *api.Client, teamName string, webSockets ...*api.WSClient) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(tui.Indigo)

	model := Model{
		client:   client,
		teamName: teamName,
		spinner:  s,
		loading:  true,
	}
	if len(webSockets) > 0 {
		model.ws = webSockets[0]
		model.wsState = api.WSState{State: api.StateReconnecting}
	}
	return model
}

func (m Model) Init() tea.Cmd {
	commands := []tea.Cmd{
		m.spinner.Tick,
		m.fetchDashboard(),
	}
	if m.ws != nil {
		commands = append(commands, waitLiveEvent(m.ws), waitLiveState(m.ws))
	}
	return tea.Batch(commands...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, m.fetchDashboard()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case refreshMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.data = msg.data
		}
		return m, scheduleRefresh()

	case tickMsg:
		m.loading = true
		return m, m.fetchDashboard()

	case liveEventMsg:
		m.loading = true
		return m, tea.Batch(m.fetchDashboard(), waitLiveEvent(m.ws))

	case liveStateMsg:
		m.wsState = msg.state
		return m, waitLiveState(m.ws)

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	header := tui.Title.Render(fmt.Sprintf("Dashboard: %s", m.teamName))
	b.WriteString(header + "\n\n")

	if m.loading && m.data == nil {
		b.WriteString(m.spinner.View() + " Loading...\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(tui.Error.Render(fmt.Sprintf("Error: %v", m.err)) + "\n")
		return b.String()
	}

	if m.data == nil {
		return b.String()
	}

	b.WriteString(tui.Bold.Render("Servers") + "\n\n")
	if len(m.data.Servers) > 0 {
		b.WriteString(m.renderServers())
	} else {
		b.WriteString(tui.Dim.Render("  No servers") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(tui.Bold.Render("Recent Deployments") + "\n\n")
	if len(m.data.RecentActivity) > 0 {
		b.WriteString(m.renderActivity())
	} else {
		b.WriteString(tui.Dim.Render("  No recent activity") + "\n")
	}

	b.WriteString("\n")

	liveState := "polling"
	if m.ws != nil {
		liveState = string(m.wsState.State)
		if liveState == "" {
			liveState = "connecting"
		}
	}
	statusLine := tui.Dim.Render(fmt.Sprintf("%s · q quit  r refresh", liveState))
	if m.loading {
		statusLine = m.spinner.View() + " refreshing...  " + statusLine
	}
	b.WriteString(statusLine)

	return b.String()
}

func (m Model) renderServers() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().Padding(0, 1)
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	var rows [][]string
	for _, s := range m.data.Servers {
		statusStyle := tui.StatusStyle(s.Status)
		rows = append(rows, []string{
			s.Name,
			s.Provider,
			statusStyle.Render("● " + s.Status),
			fmt.Sprintf("%d", s.SitesCount),
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		Headers("Name", "Provider", "Status", "Sites").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	return t.String() + "\n"
}

func (m Model) renderActivity() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().Padding(0, 1)
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	var rows [][]string
	for _, a := range m.data.RecentActivity {
		statusStyle := tui.StatusStyle(a.Status)

		commit := ""
		if a.CommitMessage != "" {
			commit = truncate(a.CommitMessage, 30)
		}

		rows = append(rows, []string{
			a.SiteName,
			a.ServerName,
			statusStyle.Render("● " + a.Status),
			commit,
			a.CreatedAt,
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		Headers("Site", "Server", "Status", "Commit", "Time").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	return t.String() + "\n"
}

func (m Model) fetchDashboard() tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.GetDashboard()
		return refreshMsg{data: data, err: err}
	}
}

func scheduleRefresh() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func waitLiveEvent(ws *api.WSClient) tea.Cmd {
	return func() tea.Msg {
		if _, err := ws.ReadMessage(); err != nil {
			return liveStateMsg{state: api.WSState{State: api.StateClosed, Err: err}}
		}
		return liveEventMsg{}
	}
}

func waitLiveState(ws *api.WSClient) tea.Cmd {
	return func() tea.Msg {
		return liveStateMsg{state: <-ws.States()}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
