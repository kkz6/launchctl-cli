package sites

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

var serverIDFlag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List sites on a server",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(serverIDFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		sites, err := client.ListSites(serverID)
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
			deployStatus := ""
			if s.LatestDeployment != nil {
				deployStatus = s.LatestDeployment.Status
			}

			rows = append(rows, []string{
				s.ID,
				s.Address,
				s.TypeLabel(),
				s.RepositoryBranch,
				output.StatusDot(s.Status),
				deployStatus,
				s.PHPVersion,
			})
		}

		output.RenderTable("Sites", []string{"ID", "Address", "Type", "Branch", "Status", "Deploy", "PHP"}, rows)
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&serverIDFlag, "server", "", "Server ID (required)")
}
