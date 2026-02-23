package deploy

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
	Use:   "list <site-id>",
	Short: "List deployments for a site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(listServerFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		deployments, err := client.ListDeployments(serverID, args[0])
		if err != nil {
			return fmt.Errorf("failed to list deployments: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(deployments, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		var rows [][]string
		for _, d := range deployments {
			commit := d.ShortGitHash
			message := ""
			if d.CommitData != nil {
				message = truncate(d.CommitData.Message, 40)
			}

			rows = append(rows, []string{
				d.ID,
				output.StatusDot(d.Status),
				commit,
				message,
				d.CreatedAt,
			})
		}

		output.RenderTable("Deployments", []string{"ID", "Status", "Commit", "Message", "Created"}, rows)
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listServerFlag, "server", "", "Server ID (required)")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
