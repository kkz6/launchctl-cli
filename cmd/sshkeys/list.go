package sshkeys

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
	Short: "List team SSH keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		keys, err := client.ListSSHKeys()
		if err != nil {
			return fmt.Errorf("failed to list SSH keys: %w", err)
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

		output.RenderTable("SSH Keys", []string{"ID", "Name", "Fingerprint", "Global"}, rows)
		return nil
	},
}
