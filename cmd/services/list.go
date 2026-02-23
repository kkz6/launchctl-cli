package services

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
	Short: "List installed services on a server",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(listServerFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		services, err := client.ListServices(serverID)
		if err != nil {
			return fmt.Errorf("failed to list services: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(services, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		var rows [][]string
		for _, s := range services {
			version := ""
			if s.Version != nil {
				version = *s.Version
			}

			rows = append(rows, []string{
				s.Name,
				s.TypeLabel,
				version,
				output.StatusDot(s.Status),
			})
		}

		output.RenderTable("Services", []string{"Name", "Type", "Version", "Status"}, rows)
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listServerFlag, "server", "", "Server ID")
}
