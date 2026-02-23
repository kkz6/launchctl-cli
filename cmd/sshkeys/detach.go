package sshkeys

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var detachServerFlag string

var detachCmd = &cobra.Command{
	Use:   "detach <key-id>",
	Short: "Detach an SSH key from a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(detachServerFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		err = client.DetachSSHKey(serverID, args[0])
		if err != nil {
			return fmt.Errorf("failed to detach SSH key: %w", err)
		}

		fmt.Println(tui.Success.Render("SSH key detached from server"))
		return nil
	},
}

func init() {
	detachCmd.Flags().StringVar(&detachServerFlag, "server", "", "Server ID")
}
