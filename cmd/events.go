package cmd

import (
	"encoding/json"
	"fmt"
	"os/signal"
	"path"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	launchapi "github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui/live"
	"github.com/spf13/cobra"
)

var (
	eventChannels []string
	eventFilters  []string
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Watch live team and resource events",
	Long:  "Open a reconnecting event console. By default it receives the active team's events; add resource channels with --channel.",
	Example: `  lctl events
  lctl events --filter 'deployment.*' --filter 'task.*'
  lctl events --channel server.01ABC --channel task.01XYZ
  lctl events --json | jq -c 'select(.event == "server.updated")'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cfg.IsAuthenticated() || cfg.TeamID == "" {
			return fmt.Errorf("not authenticated or no active team — run `lctl login` first")
		}
		client := launchapi.NewClient(cfg)
		token, err := client.ExchangeToken()
		if err != nil {
			return fmt.Errorf("authenticate event stream: %w", err)
		}
		ws, err := launchapi.NewWSClient(cfg, token)
		if err != nil {
			return err
		}
		defer ws.Close()
		for _, channel := range eventChannels {
			if err := ws.Subscribe(channel); err != nil {
				return err
			}
		}

		filter := buildEventFilter(eventFilters)
		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			for {
				select {
				case <-ctx.Done():
					return nil
				case message := <-ws.Events():
					if filter != nil && !filter(message) {
						continue
					}
					data, _ := json.Marshal(message)
					fmt.Println(string(data))
				}
			}
		}

		model := live.NewModel(live.Options{
			Title:    "Live events",
			Subtitle: fmt.Sprintf("team.%s · %s", cfg.TeamID, client.BaseURL()),
			Filter:   filter,
			WS:       ws,
		})
		_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
		return err
	},
}

func buildEventFilter(patterns []string) func(*launchapi.WSMessage) bool {
	if len(patterns) == 0 {
		return nil
	}
	cleaned := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		for _, part := range strings.Split(pattern, ",") {
			if part = strings.TrimSpace(part); part != "" {
				cleaned = append(cleaned, part)
			}
		}
	}
	return func(message *launchapi.WSMessage) bool {
		for _, pattern := range cleaned {
			if matched, _ := path.Match(pattern, message.Event); matched {
				return true
			}
		}
		return false
	}
}

func init() {
	eventsCmd.Flags().StringSliceVar(&eventChannels, "channel", nil, "Additional resource channel (repeatable)")
	eventsCmd.Flags().StringSliceVar(&eventFilters, "filter", nil, "Event glob such as deployment.* (repeatable)")
	rootCmd.AddCommand(eventsCmd)
}
