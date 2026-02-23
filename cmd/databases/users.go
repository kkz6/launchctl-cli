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

var usersServerFlag string

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "List database users on a server",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(usersServerFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		users, err := client.ListDatabaseUsers(serverID)
		if err != nil {
			return fmt.Errorf("failed to list database users: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(users, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		var rows [][]string
		for _, u := range users {
			var dbNames []string
			for _, db := range u.Databases {
				dbNames = append(dbNames, db.Name)
			}

			rows = append(rows, []string{
				u.ID,
				u.Name,
				u.Host,
				output.StatusDot(u.Status),
				strings.Join(dbNames, ", "),
			})
		}

		output.RenderTable("Database Users", []string{"ID", "Name", "Host", "Status", "Databases"}, rows)
		return nil
	},
}

func init() {
	usersCmd.Flags().StringVar(&usersServerFlag, "server", "", "Server ID")
}
