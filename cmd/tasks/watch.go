package tasks

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui/live"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch <task-id>",
	Short: "Watch task output and progress in real time",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(serverFlag)
		if err != nil {
			return err
		}
		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)
		task, err := client.GetTask(serverID, args[0])
		if err != nil {
			return fmt.Errorf("get task: %w", err)
		}
		token, err := client.ExchangeToken()
		if err != nil {
			return fmt.Errorf("authenticate task stream: %w", err)
		}
		ws, err := api.NewWSClient(cfg, token)
		if err != nil {
			return err
		}
		defer ws.Close()
		channel := "task." + task.ID
		if err := ws.Subscribe(channel); err != nil {
			return err
		}

		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			for {
				message, err := ws.ReadMessage()
				if err != nil {
					return err
				}
				if message.Channel != channel {
					continue
				}
				data, _ := json.Marshal(message)
				fmt.Println(string(data))
				if message.Event == "task.finished" || message.Event == "task.failed" || message.Event == "task.timeout" {
					return nil
				}
			}
		}

		initial := []string{fmt.Sprintf("Current status: %s", task.Status)}
		if task.Output != nil && strings.TrimSpace(*task.Output) != "" {
			initial = append(initial, strings.Split(strings.TrimRight(*task.Output, "\n"), "\n")...)
		}
		model := live.NewModel(live.Options{
			Title:        "Task · " + task.Name,
			Subtitle:     fmt.Sprintf("%s · server %s", task.ID, serverID),
			InitialLines: initial,
			Filter: func(message *api.WSMessage) bool {
				return message.Channel == channel
			},
			WS: ws,
		})
		_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
		return err
	},
}
