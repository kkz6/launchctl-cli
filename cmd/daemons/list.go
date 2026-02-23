package daemons

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

var listServerFlag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List daemons",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(listServerFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		daemons, err := client.ListDaemons(serverID)
		if err != nil {
			return fmt.Errorf("failed to list daemons: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(daemons, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		var rows [][]string
		for _, d := range daemons {
			running := "No"
			if d.Running {
				running = "Yes"
			}

			rows = append(rows, []string{
				d.ID,
				d.Command,
				d.User,
				fmt.Sprintf("%d", d.Processes),
				output.StatusDot(runningStatus(running)),
			})
		}

		output.RenderTable("Daemons", []string{"ID", "Command", "User", "Processes", "Running"}, rows)
		return nil
	},
}

func runningStatus(running string) string {
	if running == "Yes" {
		return "running"
	}
	return "stopped"
}

func init() {
	listCmd.Flags().StringVar(&listServerFlag, "server", "", "Server ID")
}
