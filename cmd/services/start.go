package services

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var startServerFlag string

var startCmd = &cobra.Command{
	Use:   "start <service-id>",
	Short: "Start a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(startServerFlag)
		if err != nil {
			return err
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		err = client.ServiceOperation(serverID, args[0], api.ServiceOperationRequest{
			Operation: "start",
		})
		if err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}

		fmt.Println(tui.Success.Render("Service start initiated"))
		return nil
	},
}

func init() {
	startCmd.Flags().StringVar(&startServerFlag, "server", "", "Server ID")
}
