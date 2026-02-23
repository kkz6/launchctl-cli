package cron

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
	Short: "List cron jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(listServerFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		crons, err := client.ListCrons(serverID)
		if err != nil {
			return fmt.Errorf("failed to list cron jobs: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(crons, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		var rows [][]string
		for _, c := range crons {
			installed := "No"
			if c.IsInstalled {
				installed = "Yes"
			}

			rows = append(rows, []string{
				c.ID,
				c.User,
				c.Expression,
				c.Command,
				installed,
			})
		}

		output.RenderTable("Cron Jobs", []string{"ID", "User", "Expression", "Command", "Installed"}, rows)
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listServerFlag, "server", "", "Server ID")
}
