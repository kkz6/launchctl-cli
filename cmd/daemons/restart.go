package daemons

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var restartServerFlag string

var restartCmd = &cobra.Command{
	Use:   "restart <daemon-id>",
	Short: "Restart a daemon",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(restartServerFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		err = client.RestartDaemon(serverID, args[0])
		if err != nil {
			return fmt.Errorf("failed to restart daemon: %w", err)
		}

		fmt.Println(tui.Success.Render("Daemon restart initiated"))
		return nil
	},
}

func init() {
	restartCmd.Flags().StringVar(&restartServerFlag, "server", "", "Server ID")
}
