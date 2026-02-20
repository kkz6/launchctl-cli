package services

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var restartServerFlag string

var restartCmd = &cobra.Command{
	Use:   "restart <service-id>",
	Short: "Restart a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(restartServerFlag)
		if err != nil {
			return err
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		err = client.ServiceOperation(serverID, args[0], api.ServiceOperationRequest{
			Operation: "restart",
		})
		if err != nil {
			return fmt.Errorf("failed to restart service: %w", err)
		}

		fmt.Println(tui.Success.Render("Service restart initiated"))
		return nil
	},
}

func init() {
	restartCmd.Flags().StringVar(&restartServerFlag, "server", "", "Server ID")
}
