package databases

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

var listServerFlag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List databases on a server",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(listServerFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		databases, err := client.ListDatabases(serverID)
		if err != nil {
			return fmt.Errorf("failed to list databases: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(databases, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		var rows [][]string
		for _, db := range databases {
			var userNames []string
			for _, u := range db.Users {
				userNames = append(userNames, u.Name)
			}

			rows = append(rows, []string{
				db.ID,
				db.Name,
				output.StatusDot(db.Status),
				strings.Join(userNames, ", "),
			})
		}

		output.RenderTable("Databases", []string{"ID", "Name", "Status", "Users"}, rows)
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listServerFlag, "server", "", "Server ID")
}
