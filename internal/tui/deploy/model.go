package deploy

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/notify"
	"github.com/kkz6/launchctl/internal/tui"
)

type statusMsg struct {
	status string
	step   string
}

type doneMsg struct {
	status string
}

type wsErrorMsg struct {
	err error
}

type taskIDFoundMsg struct {
	taskID string
}

type logsWSConnectedMsg struct {
	ws *api.LogsWSClient
}

type taskLogLineMsg struct {
	line string
}

type taskLogEndMsg struct{}

type pollTickMsg struct{}

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

type Opts struct {
	SiteName     string
	ServerID     string
	SiteID       string
	DeploymentID string
	Client       *api.Client
	JWT          string
	TeamID       string
	APIURL       string
	WS           *api.WSClient
}

type Model struct {
	viewport     viewport.Model
	spinner      spinner.Model
	logs         []string
	status       string
	step         string
	done         bool
	err          error
	siteName     string
	width        int
	height       int
	ws           *api.WSClient
	logsWS       *api.LogsWSClient
	client       *api.Client
	jwt          string
	teamID       string
	apiURL       string
	serverID     string
	siteID       string
	deploymentID string
	taskID       string
}

func NewModel(opts Opts) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(tui.Indigo)

	vp := viewport.New(80, 20)

	return Model{
		viewport:     vp,
		spinner:      s,
		status:       "pending",
		siteName:     opts.SiteName,
		ws:           opts.WS,
		client:       opts.Client,
		jwt:          opts.JWT,
		teamID:       opts.TeamID,
		apiURL:       opts.APIURL,
		serverID:     opts.ServerID,
		siteID:       opts.SiteID,
		deploymentID: opts.DeploymentID,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		listenEventWS(m.ws, m.deploymentID),
		pollForTaskID(m.client, m.serverID, m.siteID, m.deploymentID),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if !m.done {
				notify.Info(fmt.Sprintf("Deployment of %s running in background", m.siteName))
				startBackgroundWatcher(m.client, m.jwt, m.serverID, m.siteID, m.deploymentID, m.siteName)
			}
			return m, tea.Quit
		case "l":
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 8
		m.viewport.Width = msg.Width - 2
		m.viewport.Height = msg.Height - headerHeight
		if m.viewport.Height < 3 {
			m.viewport.Height = 3
		}
		m.viewport.SetContent(m.renderLogs())

	case taskIDFoundMsg:
		m.taskID = msg.taskID
		cmds = append(cmds, connectLogsWS(m.jwt, m.teamID, m.serverID, msg.taskID, m.apiURL))

	case logsWSConnectedMsg:
		m.logsWS = msg.ws
		cmds = append(cmds, listenLogsWS(msg.ws))

	case taskLogLineMsg:
		m.logs = append(m.logs, msg.line)
		m.viewport.SetContent(m.renderLogs())
		m.viewport.GotoBottom()
		if m.logsWS != nil {
			cmds = append(cmds, listenLogsWS(m.logsWS))
		}

	case taskLogEndMsg:
		// Logs stream ended, task output complete

	case statusMsg:
		m.status = msg.status
		m.step = msg.step
		if !m.done {
			cmds = append(cmds, listenEventWS(m.ws, m.deploymentID))
		}

	case doneMsg:
		m.done = true
		m.status = msg.status
		if msg.status == "finished" {
			notify.Success(fmt.Sprintf("Deployment of %s finished", m.siteName))
		} else {
			notify.Error(fmt.Sprintf("Deployment of %s %s", m.siteName, msg.status))
		}

	case wsErrorMsg:
		if !m.done {
			m.err = msg.err
			m.done = true
			notify.Error(fmt.Sprintf("Deployment error: %s", msg.err))
		}

	case pollTickMsg:
		if m.taskID == "" && !m.done {
			cmds = append(cmds, pollForTaskID(m.client, m.serverID, m.siteID, m.deploymentID))
		}

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
		return logLineStyle.Render("Waiting for output...")
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

	title := tui.Title.Render(fmt.Sprintf("Deploying: %s", m.siteName))
	b.WriteString(title + "\n")

	var statusLine string
	if m.done {
		statusStyle := tui.StatusStyle(m.status)
		icon := "\u2713"
		if m.status == "failed" || m.status == "timeout" || m.status == "cancelled" {
			icon = "\u2717"
		}
		statusLine = statusStyle.Render(fmt.Sprintf("%s  %s", icon, m.status))
	} else {
		statusLine = m.spinner.View() + " " + tui.Value.Render(m.status)
		if m.step != "" {
			statusLine += "  " + stepStyle.Render(m.step)
		}
	}
	b.WriteString(statusBarStyle.Render(statusLine) + "\n")

	logHeader := logHeaderStyle.Render(fmt.Sprintf("Output  %s",
		logCountStyle.Render(fmt.Sprintf("(%d lines)", len(m.logs)))))
	b.WriteString(logHeader + "\n")

	logBox := logBorderStyle.Width(w - 2)
	b.WriteString(logBox.Render(m.viewport.View()) + "\n")

	b.WriteString("\n")
	if m.done {
		b.WriteString(tui.Dim.Render("q quit  \u2191/\u2193 scroll"))
	} else {
		b.WriteString(tui.Dim.Render("q background  \u2191/\u2193 scroll"))
	}

	return b.String()
}

func listenEventWS(ws *api.WSClient, deploymentID string) tea.Cmd {
	return func() tea.Msg {
		msg, err := ws.ReadMessage()
		if err != nil {
			return wsErrorMsg{err: err}
		}

		switch msg.Event {
		case "deployment.progress":
			var event api.DeploymentLogEvent
			if err := parseEventData(msg.Data, &event); err == nil {
				if event.DeploymentID != deploymentID {
					return listenEventWS(ws, deploymentID)()
				}
				return statusMsg{status: event.Status, step: event.Step}
			}

		case "deployment.finished", "deployment.failed", "deployment.timeout", "deployment.cancelled", "deployment.started":
			var event api.DeploymentLogEvent
			if err := parseEventData(msg.Data, &event); err == nil {
				if event.DeploymentID != deploymentID {
					return listenEventWS(ws, deploymentID)()
				}
				switch msg.Event {
				case "deployment.started":
					return statusMsg{status: "deploying", step: event.Step}
				case "deployment.finished":
					return doneMsg{status: "finished"}
				case "deployment.failed":
					return doneMsg{status: "failed"}
				case "deployment.timeout":
					return doneMsg{status: "timeout"}
				default:
					return doneMsg{status: "cancelled"}
				}
			}
		}

		return listenEventWS(ws, deploymentID)()
	}
}

func pollForTaskID(client *api.Client, serverID, siteID, deploymentID string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)

		dep, err := client.GetDeployment(serverID, siteID, deploymentID)
		if err != nil {
			return pollTickMsg{}
		}

		if dep.TaskID != nil && *dep.TaskID != "" {
			return taskIDFoundMsg{taskID: *dep.TaskID}
		}

		return pollTickMsg{}
	}
}

func connectLogsWS(jwt, teamID, serverID, taskID, apiURL string) tea.Cmd {
	return func() tea.Msg {
		ws, err := api.NewLogsWSClientDirect(jwt, teamID, serverID, "task", taskID, apiURL)
		if err != nil {
			return taskLogEndMsg{}
		}
		return logsWSConnectedMsg{ws: ws}
	}
}

func listenLogsWS(ws *api.LogsWSClient) tea.Cmd {
	return func() tea.Msg {
		line, err := ws.ReadMessage()
		if err != nil {
			return taskLogEndMsg{}
		}
		return taskLogLineMsg{line: line}
	}
}

// startBackgroundWatcher starts a goroutine that watches for deployment completion
// and pushes notifications when done.
func startBackgroundWatcher(client *api.Client, jwt, serverID, siteID, deploymentID, siteName string) {
	go func() {
		for i := 0; i < 120; i++ { // poll for up to ~4 minutes
			time.Sleep(2 * time.Second)
			dep, err := client.GetDeployment(serverID, siteID, deploymentID)
			if err != nil {
				continue
			}
			switch dep.Status {
			case "finished":
				notify.Success(fmt.Sprintf("Deployment of %s finished", siteName))
				return
			case "failed":
				notify.Error(fmt.Sprintf("Deployment of %s failed", siteName))
				return
			case "timeout":
				notify.Error(fmt.Sprintf("Deployment of %s timed out", siteName))
				return
			case "cancelled":
				notify.Warn(fmt.Sprintf("Deployment of %s cancelled", siteName))
				return
			}
		}
	}()
}

func parseEventData(data []byte, v any) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	return api.UnmarshalEventData(data, v)
}
