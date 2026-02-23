package servers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var metricsWatchFlag bool

var metricsCmd = &cobra.Command{
	Use:   "metrics <id>",
	Short: "Show latest server metrics",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		server, err := client.GetServer(args[0])
		if err != nil {
			return fmt.Errorf("failed to get server: %w", err)
		}

		if metricsWatchFlag {
			return runMetricsWatch(client, server)
		}

		metrics, err := client.GetServerMetrics(args[0])
		if err != nil {
			return fmt.Errorf("failed to get metrics: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(metrics, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		printMetrics(server.Name, metrics)
		return nil
	},
}

func init() {
	metricsCmd.Flags().BoolVarP(&metricsWatchFlag, "watch", "w", false, "Watch metrics in real-time")
}

func printMetrics(name string, metrics *api.MetricResponse) {
	fmt.Println()
	fmt.Println(tui.Title.Render(fmt.Sprintf("Metrics: %s", name)))
	fmt.Println()
	fmt.Println(tui.Label.Render("Load:") + tui.Value.Render(fmt.Sprintf("%.2f", metrics.Load)))
	fmt.Println()
	fmt.Println(tui.Label.Render("Memory:") + renderBar(metrics.MemoryUsagePercent))
	fmt.Println(tui.Label.Render("") + tui.Dim.Render(fmt.Sprintf("%.0f MB / %.0f MB (%.1f%%)",
		metrics.MemoryUsed/1024/1024, metrics.MemoryTotal/1024/1024, metrics.MemoryUsagePercent)))
	fmt.Println()
	fmt.Println(tui.Label.Render("Disk:") + renderBar(metrics.DiskUsagePercent))
	fmt.Println(tui.Label.Render("") + tui.Dim.Render(fmt.Sprintf("%.1f GB / %.1f GB (%.1f%%)",
		metrics.DiskUsed/1024/1024/1024, metrics.DiskTotal/1024/1024/1024, metrics.DiskUsagePercent)))
	fmt.Println()
	fmt.Println(tui.Label.Render("Updated:") + tui.Dim.Render(metrics.CreatedAt))
	fmt.Println()
}

func renderBar(percent float64) string {
	width := 30
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}

	color := tui.Green
	if percent > 80 {
		color = tui.Red
	} else if percent > 60 {
		color = tui.Yellow
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += tui.Bold.Foreground(color).Render("█")
		} else {
			bar += tui.Dim.Render("░")
		}
	}

	return bar + fmt.Sprintf(" %.1f%%", percent)
}

// Real-time metrics TUI

type metricsModel struct {
	client     *api.Client
	serverID   string
	serverName string
	metrics    *api.MetricResponse
	spinner    spinner.Model
	err        error
	width      int
	height     int
}

type metricsTickMsg time.Time
type metricsFetchedMsg struct{ metrics *api.MetricResponse }
type metricsFetchErrMsg struct{ err error }

func runMetricsWatch(client *api.Client, server *api.ServerResponse) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(tui.Indigo)

	m := metricsModel{
		client:     client,
		serverID:   server.ID,
		serverName: server.Name,
		spinner:    s,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m metricsModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchMetrics())
}

func (m metricsModel) fetchMetrics() tea.Cmd {
	return func() tea.Msg {
		metrics, err := m.client.GetServerMetrics(m.serverID)
		if err != nil {
			return metricsFetchErrMsg{err: err}
		}
		return metricsFetchedMsg{metrics: metrics}
	}
}

func tickMetrics() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return metricsTickMsg(t)
	})
}

func (m metricsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, m.fetchMetrics()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case metricsFetchedMsg:
		m.metrics = msg.metrics
		m.err = nil
		return m, tickMetrics()

	case metricsFetchErrMsg:
		m.err = msg.err
		return m, tickMetrics()

	case metricsTickMsg:
		return m, m.fetchMetrics()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m metricsModel) View() string {
	var b strings.Builder

	title := tui.Title.Render(fmt.Sprintf("  Metrics: %s", m.serverName))
	b.WriteString(title + "\n\n")

	if m.err != nil {
		b.WriteString(tui.Error.Render(fmt.Sprintf("  Error: %s", m.err)) + "\n")
		b.WriteString(tui.Dim.Render("  Retrying...") + "\n")
	}

	if m.metrics == nil {
		b.WriteString("  " + m.spinner.View() + " Fetching metrics...\n")
	} else {
		metrics := m.metrics

		loadColor := tui.Green
		if metrics.Load > 4 {
			loadColor = tui.Red
		} else if metrics.Load > 2 {
			loadColor = tui.Yellow
		}

		b.WriteString(fmt.Sprintf("  %s %s\n\n",
			tui.Label.Render("Load:"),
			lipgloss.NewStyle().Foreground(loadColor).Bold(true).Render(fmt.Sprintf("%.2f", metrics.Load)),
		))

		b.WriteString(fmt.Sprintf("  %s %s\n", tui.Label.Render("Memory:"), renderWideBar(metrics.MemoryUsagePercent, m.width-30)))
		b.WriteString(fmt.Sprintf("  %s %s\n\n",
			tui.Label.Render(""),
			tui.Dim.Render(fmt.Sprintf("%.0f MB / %.0f MB (%.1f%%)",
				metrics.MemoryUsed/1024/1024, metrics.MemoryTotal/1024/1024, metrics.MemoryUsagePercent)),
		))

		b.WriteString(fmt.Sprintf("  %s %s\n", tui.Label.Render("Disk:"), renderWideBar(metrics.DiskUsagePercent, m.width-30)))
		b.WriteString(fmt.Sprintf("  %s %s\n\n",
			tui.Label.Render(""),
			tui.Dim.Render(fmt.Sprintf("%.1f GB / %.1f GB (%.1f%%)",
				metrics.DiskUsed/1024/1024/1024, metrics.DiskTotal/1024/1024/1024, metrics.DiskUsagePercent)),
		))

		b.WriteString(fmt.Sprintf("  %s %s\n",
			tui.Label.Render("Updated:"),
			tui.Dim.Render(metrics.CreatedAt),
		))
	}

	b.WriteString("\n")
	b.WriteString(tui.Dim.Render("  q quit  r refresh  (auto-refreshes every 5s)"))

	return b.String()
}

func renderWideBar(percent float64, width int) string {
	if width < 10 {
		width = 30
	}
	if width > 60 {
		width = 60
	}

	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}

	color := tui.Green
	if percent > 80 {
		color = tui.Red
	} else if percent > 60 {
		color = tui.Yellow
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += tui.Bold.Foreground(color).Render("█")
		} else {
			bar += tui.Dim.Render("░")
		}
	}

	return bar + fmt.Sprintf(" %.1f%%", percent)
}
