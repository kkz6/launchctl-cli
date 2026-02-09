package servers

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		servers, _, err := client.ListServers()
		if err != nil {
			return fmt.Errorf("failed to list servers: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(servers, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		var rows [][]string
		for _, s := range servers {
			ip := ""
			if s.PublicIPv4 != nil {
				ip = *s.PublicIPv4
			}

			rows = append(rows, []string{
				s.ID,
				s.Name,
				s.ProviderLabel,
				output.StatusDot(s.Status),
				ip,
			})
		}

		output.RenderTable("Servers", []string{"ID", "Name", "Provider", "Status", "IP"}, rows)
		return nil
	},
}
