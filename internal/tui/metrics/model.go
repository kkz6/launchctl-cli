package metrics

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui"
)

type streamEvent struct {
	Event   string `json:"event"`
	Message string `json:"message,omitempty"`
}

type SystemInfo struct {
	Hostname    string `json:"hostname"`
	OS          string `json:"os"`
	Kernel      string `json:"kernel"`
	CPUModel    string `json:"cpu_model"`
	CPUCores    int    `json:"cpu_cores"`
	TotalMemory int64  `json:"total_memory"`
	Uptime      int    `json:"uptime"`
}

type Metrics struct {
	Timestamp string      `json:"timestamp"`
	CPU       float64     `json:"cpu"`
	Load      []float64   `json:"load"`
	Memory    MemoryInfo  `json:"memory"`
	Disk      DiskInfo    `json:"disk"`
	Processes []Process   `json:"processes"`
	Network   NetworkInfo `json:"network"`
}

type MemoryInfo struct {
	Total   float64 `json:"total"`
	Used    float64 `json:"used"`
	Free    float64 `json:"free"`
	Percent float64 `json:"percent"`
}

type DiskInfo struct {
	Total   float64 `json:"total"`
	Used    float64 `json:"used"`
	Free    float64 `json:"free"`
	Percent float64 `json:"percent"`
}

type Process struct {
	PID     int     `json:"pid"`
	User    string  `json:"user"`
	CPU     float64 `json:"cpu"`
	Mem     float64 `json:"mem"`
	Command string  `json:"command"`
}

type NetworkInfo struct {
	RxBytes int64 `json:"rx_bytes"`
	TxBytes int64 `json:"tx_bytes"`
	RxRate  int64 `json:"rx_rate"`
	TxRate  int64 `json:"tx_rate"`
}

type connectedMsg struct{}
type systemInfoMsg struct{ info SystemInfo }
type metricsMsg struct{ data Metrics }
type errMsg struct{ err error }

var (
	sectionLabel = lipgloss.NewStyle().
			Foreground(tui.Slate).
			Bold(true).
			Width(11)

	barLabel = lipgloss.NewStyle().
			Foreground(tui.Slate).
			Bold(true).
			Width(11)

	infoLabel = lipgloss.NewStyle().
			Foreground(tui.Slate).
			Width(11)

	infoValue = lipgloss.NewStyle().
			Foreground(tui.White)

	processHeader = lipgloss.NewStyle().
			Foreground(tui.Slate).
			Bold(true)

	processRow = lipgloss.NewStyle().
			Foreground(tui.White)
)

type Model struct {
	spinner    spinner.Model
	serverName string
	ws         *api.MetricsWSClient
	connected  bool
	sysInfo    *SystemInfo
	metrics    *Metrics
	done       bool
	err        error
	width      int
	height     int
}

func NewModel(serverName string, ws *api.MetricsWSClient) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(tui.Indigo)

	return Model{
		spinner:    s,
		serverName: serverName,
		ws:         ws,
	}
}

func Run(serverName string, ws *api.MetricsWSClient) error {
	model := NewModel(serverName, ws)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()

	return err
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
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case connectedMsg:
		m.connected = true
		cmds = append(cmds, listenWS(m.ws))

	case systemInfoMsg:
		m.sysInfo = &msg.info
		cmds = append(cmds, listenWS(m.ws))

	case metricsMsg:
		m.metrics = &msg.data
		cmds = append(cmds, listenWS(m.ws))

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

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var b strings.Builder
	w := m.width
	if w == 0 {
		w = 80
	}

	title := tui.Title.Render(fmt.Sprintf("Metrics: %s", m.serverName))
	if m.connected {
		gap := w - lipgloss.Width(title) - 12
		if gap < 2 {
			gap = 2
		}
		title += strings.Repeat(" ", gap)
		title += tui.StatusConnected.Render("\u25cf Connected")
	}
	b.WriteString(title + "\n")

	divW := w - 2
	if divW < 10 {
		divW = 10
	}
	b.WriteString(tui.Dim.Render(strings.Repeat("\u2500", divW)) + "\n\n")

	if m.err != nil {
		b.WriteString(tui.Error.Render("\u2717 "+m.err.Error()) + "\n\n")
		b.WriteString(tui.Dim.Render("q quit"))
		return b.String()
	}

	if m.metrics == nil {
		b.WriteString(m.spinner.View() + " " + tui.Dim.Render("Waiting for metrics...") + "\n\n")
		b.WriteString(tui.Dim.Render("q quit"))
		return b.String()
	}

	met := m.metrics

	b.WriteString(renderResourceBar("CPU", met.CPU, ""))
	b.WriteString(renderResourceBar("Memory", met.Memory.Percent,
		formatBytes(met.Memory.Used)+" / "+formatBytes(met.Memory.Total)))
	b.WriteString(renderResourceBar("Disk", met.Disk.Percent,
		formatBytes(met.Disk.Used)+" / "+formatBytes(met.Disk.Total)))
	b.WriteString("\n")

	if len(met.Load) >= 3 {
		b.WriteString(fmt.Sprintf("%s %s      %s      %s\n",
			sectionLabel.Render("Load"),
			infoValue.Render(fmt.Sprintf("%.2f", met.Load[0])),
			tui.Dim.Render(fmt.Sprintf("%.2f", met.Load[1])),
			tui.Dim.Render(fmt.Sprintf("%.2f", met.Load[2]))))
		b.WriteString(fmt.Sprintf("%s %s      %s     %s\n\n",
			lipgloss.NewStyle().Width(11).Render(""),
			tui.Dim.Render("1 min"),
			tui.Dim.Render("5 min"),
			tui.Dim.Render("15 min")))
	}

	if len(met.Processes) > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(tui.Indigo).Bold(true).Render("Top Processes") + "\n")
		header := fmt.Sprintf(" %-7s %-14s %6s  %6s   %-s", "PID", "User", "CPU%", "Mem%", "Command")
		b.WriteString(processHeader.Render(header) + "\n")

		for _, p := range met.Processes {
			row := fmt.Sprintf(" %-7d %-14s %5.1f   %5.1f   %s",
				p.PID, truncate(p.User, 14), p.CPU, p.Mem, truncate(p.Command, 35))
			b.WriteString(processRow.Render(row) + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("%s %s %s    %s %s\n\n",
		sectionLabel.Render("Network"),
		tui.StatusConnected.Render("\u2193"),
		infoValue.Render(formatRate(met.Network.RxRate)),
		tui.StatusPending.Render("\u2191"),
		infoValue.Render(formatRate(met.Network.TxRate))))

	if m.sysInfo != nil {
		info := m.sysInfo
		b.WriteString(lipgloss.NewStyle().Foreground(tui.Indigo).Bold(true).Render("System") + "\n")

		cpuInfo := info.CPUModel
		if info.CPUCores > 0 {
			cpuInfo += fmt.Sprintf(" (%d cores)", info.CPUCores)
		}

		renderInfoRow := func(l1, v1, l2, v2 string) {
			left := infoLabel.Render(l1) + infoValue.Render(v1)
			right := infoLabel.Render(l2) + infoValue.Render(v2)
			gap := 32 - lipgloss.Width(left)
			if gap < 2 {
				gap = 2
			}
			b.WriteString(left + strings.Repeat(" ", gap) + right + "\n")
		}

		renderInfoRow("Hostname", info.Hostname, "OS", info.OS)
		renderInfoRow("Kernel", info.Kernel, "CPU", truncate(cpuInfo, 30))
		renderInfoRow("Memory", formatBytes(float64(info.TotalMemory)), "Uptime", formatUptime(info.Uptime))
	}

	b.WriteString("\n")
	footer := tui.Dim.Render("q quit")
	if met.Timestamp != "" {
		ts := met.Timestamp
		if idx := strings.Index(ts, "T"); idx >= 0 {
			ts = strings.TrimSuffix(ts[idx+1:], "Z")
		}
		spacing := w - 10 - len(ts)
		if spacing < 2 {
			spacing = 2
		}
		footer += strings.Repeat(" ", spacing)
		footer += tui.Dim.Render(ts)
	}
	b.WriteString(footer)

	return b.String()
}

func listenWS(ws *api.MetricsWSClient) tea.Cmd {
	return func() tea.Msg {
		data, err := ws.ReadRaw()
		if err != nil {
			return errMsg{err: fmt.Errorf("connection closed")}
		}

		var evt streamEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return errMsg{err: fmt.Errorf("invalid event: %w", err)}
		}

		switch evt.Event {
		case "connected":
			return connectedMsg{}

		case "system_info":
			var info SystemInfo
			if err := json.Unmarshal(data, &info); err != nil {
				return errMsg{err: err}
			}
			return systemInfoMsg{info: info}

		case "metrics":
			var met Metrics
			if err := json.Unmarshal(data, &met); err != nil {
				return errMsg{err: err}
			}
			return metricsMsg{data: met}

		case "error":
			return errMsg{err: fmt.Errorf(evt.Message)}

		default:
			return listenWS(ws)()
		}
	}
}

func renderResourceBar(label string, percent float64, detail string) string {
	barWidth := 30
	filled := int(percent / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}

	color := tui.Green
	if percent > 80 {
		color = tui.Red
	} else if percent > 60 {
		color = tui.Yellow
	}

	var bar strings.Builder
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar.WriteString(lipgloss.NewStyle().Foreground(color).Bold(true).Render("\u2588"))
		} else {
			bar.WriteString(tui.Dim.Render("\u2591"))
		}
	}

	result := barLabel.Render(label) + bar.String() + "  " + fmt.Sprintf("%5.1f%%", percent)
	if detail != "" {
		result += "    " + tui.Dim.Render(detail)
	}

	return result + "\n"
}

func formatBytes(b float64) string {
	if b <= 0 {
		return "0 B"
	}
	if b < 1024 {
		return fmt.Sprintf("%.0f B", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.1f KB", b/1024)
	}
	if b < 1024*1024*1024 {
		return fmt.Sprintf("%.0f MB", b/(1024*1024))
	}

	return fmt.Sprintf("%.2f GB", b/(1024*1024*1024))
}

func formatRate(bytesPerSec int64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%d B/s", bytesPerSec)
	}
	if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", float64(bytesPerSec)/1024)
	}

	return fmt.Sprintf("%.1f MB/s", float64(bytesPerSec)/(1024*1024))
}

func formatUptime(secs int) string {
	days := secs / 86400
	hours := (secs % 86400) / 3600
	mins := (secs % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}

	return fmt.Sprintf("%dm", mins)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
