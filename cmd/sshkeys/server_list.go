package sshkeys

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

var serverListServerFlag string

var serverListCmd = &cobra.Command{
	Use:   "server-list",
	Short: "List SSH keys attached to a server",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(serverListServerFlag)
		if err != nil {
			return err
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		keys, err := client.ListServerSSHKeys(serverID)
		if err != nil {
			return fmt.Errorf("failed to list server SSH keys: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(keys, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		var rows [][]string
		for _, k := range keys {
			global := "No"
			if k.IsGlobal {
				global = "Yes"
			}

			rows = append(rows, []string{
				k.ID,
				k.Name,
				k.Fingerprint,
				global,
			})
		}

		output.RenderTable("Server SSH Keys", []string{"ID", "Name", "Fingerprint", "Global"}, rows)
		return nil
	},
}

func init() {
	serverListCmd.Flags().StringVar(&serverListServerFlag, "server", "", "Server ID")
}
