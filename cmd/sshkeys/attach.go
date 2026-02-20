package sshkeys

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var attachServerFlag string

var attachCmd = &cobra.Command{
	Use:   "attach <key-id>",
	Short: "Attach an SSH key to a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(attachServerFlag)
		if err != nil {
			return err
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		err = client.AttachSSHKey(serverID, api.AttachSSHKeyRequest{
			SSHKeyID: args[0],
		})
		if err != nil {
			return fmt.Errorf("failed to attach SSH key: %w", err)
		}

		fmt.Println(tui.Success.Render("SSH key attached to server"))
		return nil
	},
}

func init() {
	attachCmd.Flags().StringVar(&attachServerFlag, "server", "", "Server ID")
}
