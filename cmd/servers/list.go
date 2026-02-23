package servers

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := appstate.GetConfig()
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
				s.TypeLabel,
				output.StatusDot(s.Status),
				ip,
				fmt.Sprintf("%d", s.SitesCount),
			})
		}

		output.RenderTable("Servers", []string{"ID", "Name", "Provider", "Type", "Status", "IP", "Sites"}, rows)
		return nil
	},
}
