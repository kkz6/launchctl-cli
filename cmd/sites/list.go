package sites

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/spf13/cobra"
)

var serverIDFlag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List sites on a server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if serverIDFlag == "" {
			return fmt.Errorf("--server flag is required")
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		sites, err := client.ListSites(serverIDFlag)
		if err != nil {
			return fmt.Errorf("failed to list sites: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(sites, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		var rows [][]string
		for _, s := range sites {
			status := s.Status
			deployStatus := ""
			if s.LatestDeployment != nil {
				deployStatus = s.LatestDeployment.Status
			}

			rows = append(rows, []string{
				s.ID,
				s.Address,
				s.Type,
				output.StatusDot(status),
				deployStatus,
			})
		}

		output.RenderTable("Sites", []string{"ID", "Address", "Type", "Status", "Last Deploy"}, rows)
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&serverIDFlag, "server", "", "Server ID (required)")
}
