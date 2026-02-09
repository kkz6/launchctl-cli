package servers

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics <id>",
	Short: "Show latest server metrics",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		server, err := client.GetServer(args[0])
		if err != nil {
			return fmt.Errorf("failed to get server: %w", err)
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

		fmt.Println()
		fmt.Println(tui.Title.Render(fmt.Sprintf("Metrics: %s", server.Name)))
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

		return nil
	},
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
