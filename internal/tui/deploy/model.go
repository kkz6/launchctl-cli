package deploy

import (
	"context"
	"errors"
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

type deploymentPolledMsg struct {
	deployment *api.DeploymentResponse
	err        error
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
			Background(tui.Panel).
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
			Foreground(tui.Slate)
)

type Opts struct {
	Context      context.Context
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
	viewport         viewport.Model
	spinner          spinner.Model
	logs             []string
	status           string
	step             string
	done             bool
	err              error
	siteName         string
	width            int
	height           int
	ws               *api.WSClient
	logsWS           *api.LogsWSClient
	client           *api.Client
	jwt              string
	teamID           string
	apiURL           string
	serverID         string
	siteID           string
	deploymentID     string
	taskID           string
	ctx              context.Context
	cancel           context.CancelFunc
	connectionNotice string
}

func NewModel(opts Opts) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(tui.Indigo)

	vp := viewport.New(80, 20)
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)

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
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		listenEventWS(m.ws, m.deploymentID),
		pollDeployment(m.ctx, m.client, m.serverID, m.siteID, m.deploymentID),
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
			m.Close()
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

	case deploymentPolledMsg:
		if m.done || m.ctx.Err() != nil {
			return m, nil
		}
		if msg.err != nil {
			m.connectionNotice = "Unable to refresh deployment status; retrying"
			var apiErr *api.APIError
			if errors.As(msg.err, &apiErr) && (apiErr.StatusCode == 401 || apiErr.StatusCode == 403 || apiErr.StatusCode == 404) {
				m.err = fmt.Errorf("cannot monitor deployment %s: %w", m.deploymentID, msg.err)
				m.done = true
				return m, nil
			}
		} else if msg.deployment != nil {
			m.connectionNotice = ""
			m.status = msg.deployment.Status
			if m.taskID == "" && msg.deployment.TaskID != nil && *msg.deployment.TaskID != "" {
				m.taskID = *msg.deployment.TaskID
				cmds = append(cmds, connectLogsWS(m.jwt, m.teamID, m.serverID, m.taskID, m.apiURL))
			}
			if terminalDeploymentStatus(m.status) {
				m.complete(m.status)
				return m, tea.Batch(cmds...)
			}
		}
		cmds = append(cmds, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return pollTickMsg{} }))

	case logsWSConnectedMsg:
		if m.ctx.Err() != nil {
			_ = msg.ws.Close()
			return m, nil
		}
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
		if m.done {
			return m, nil
		}
		m.status = msg.status
		m.step = msg.step
		if terminalDeploymentStatus(msg.status) {
			m.complete(msg.status)
			return m, nil
		}
		if !m.done {
			cmds = append(cmds, listenEventWS(m.ws, m.deploymentID))
		}

	case doneMsg:
		m.complete(msg.status)

	case wsErrorMsg:
		if !m.done {
			m.connectionNotice = "Live events disconnected; checking deployment status"
		}

	case pollTickMsg:
		if !m.done && m.ctx.Err() == nil {
			cmds = append(cmds, pollDeployment(m.ctx, m.client, m.serverID, m.siteID, m.deploymentID))
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

func (m Model) Err() error { return m.err }

func (m Model) Done() bool { return m.done }

// Close stops pending status requests and closes this view's log stream.
func (m Model) Close() {
	m.cancel()
	if m.logsWS != nil {
		_ = m.logsWS.Close()
	}
}

func (m *Model) complete(status string) {
	if m.done {
		return
	}
	m.done = true
	m.status = status
	if status == "finished" {
		notify.Success(fmt.Sprintf("Deployment of %s finished", m.siteName))
	} else {
		m.err = fmt.Errorf("deployment %s %s", m.deploymentID, status)
		notify.Error(fmt.Sprintf("Deployment of %s %s", m.siteName, status))
	}
}

func terminalDeploymentStatus(status string) bool {
	return status == "finished" || status == "failed" || status == "timeout" || status == "cancelled"
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
	if m.err != nil {
		b.WriteString(tui.Warning.Render(m.err.Error()) + "\n")
	} else if m.connectionNotice != "" {
		b.WriteString(tui.Dim.Render(m.connectionNotice) + "\n")
	}

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
	if ws == nil {
		return nil
	}
	return func() tea.Msg {
		for {
			msg, err := ws.ReadMessage()
			if err != nil {
				return wsErrorMsg{err: err}
			}

			switch msg.Event {
			case "deployment.progress":
				var event api.DeploymentLogEvent
				if err := parseEventData(msg.Data, &event); err == nil {
					if event.DeploymentID != deploymentID {
						continue
					}
					return statusMsg{status: event.Status, step: event.Step}
				}

			case "deployment.finished", "deployment.failed", "deployment.timeout", "deployment.cancelled", "deployment.started", "deployment.rollback.completed", "deployment.rollback.failed", "deployment.rollback.started":
				var event api.DeploymentLogEvent
				if err := parseEventData(msg.Data, &event); err == nil {
					if event.DeploymentID != deploymentID {
						continue
					}
					switch msg.Event {
					case "deployment.started", "deployment.rollback.started":
						return statusMsg{status: "installing", step: event.Step}
					case "deployment.finished", "deployment.rollback.completed":
						return doneMsg{status: "finished"}
					case "deployment.failed", "deployment.rollback.failed":
						return doneMsg{status: "failed"}
					case "deployment.timeout":
						return doneMsg{status: "timeout"}
					default:
						return doneMsg{status: "cancelled"}
					}
				}
			}

		}
	}
}

func pollDeployment(ctx context.Context, client *api.Client, serverID, siteID, deploymentID string) tea.Cmd {
	return func() tea.Msg {
		dep, err := client.GetDeploymentContext(ctx, serverID, siteID, deploymentID)
		return deploymentPolledMsg{deployment: dep, err: err}
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
