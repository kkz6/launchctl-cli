package tasks

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent tasks for a server",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(serverFlag)
		if err != nil {
			return err
		}
		items, err := api.NewClient(appstate.GetConfig()).ListTasks(serverID)
		if err != nil {
			return fmt.Errorf("list tasks: %w", err)
		}
		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			data, _ := json.MarshalIndent(items, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		rows := make([][]string, 0, len(items))
		for _, item := range items {
			exit := "—"
			if item.ExitCode != nil {
				exit = fmt.Sprint(*item.ExitCode)
			}
			rows = append(rows, []string{item.ID, item.Name, item.Type, item.Status, exit, item.CreatedAt})
		}
		output.RenderTable("Server Tasks", []string{"ID", "Name", "Type", "Status", "Exit", "Created"}, rows)
		return nil
	},
}
