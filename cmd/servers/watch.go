package servers

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/tui/live"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch <server-id>",
	Short: "Watch server provisioning and lifecycle events",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)
		server, err := client.GetServer(args[0])
		if err != nil {
			return fmt.Errorf("get server: %w", err)
		}
		token, err := client.ExchangeToken()
		if err != nil {
			return fmt.Errorf("authenticate server stream: %w", err)
		}
		ws, err := api.NewWSClient(cfg, token)
		if err != nil {
			return err
		}
		defer ws.Close()
		channel := "server." + server.ID
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
				if message.Event == "server.provisioned" || message.Event == "server.provision_failed" || message.Event == "server.provision_timeout" {
					return nil
				}
			}
		}

		progress := 0
		if server.Progress != nil {
			progress = *server.Progress
		}
		step := "waiting for the next update"
		if server.ProgressStep != nil && *server.ProgressStep != "" {
			step = *server.ProgressStep
		}
		model := live.NewModel(live.Options{
			Title:        "Server · " + server.Name,
			Subtitle:     fmt.Sprintf("%s · %s", server.ID, client.BaseURL()),
			InitialLines: []string{fmt.Sprintf("Current status: %s · %d%% · %s", server.Status, progress, step)},
			Filter: func(message *api.WSMessage) bool {
				return message.Channel == channel
			},
			WS: ws,
		})
		_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
		return err
	},
}
