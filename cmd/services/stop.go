package services

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var stopServerFlag string

var stopCmd = &cobra.Command{
	Use:   "stop <service-id>",
	Short: "Stop a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(stopServerFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		err = client.ServiceOperation(serverID, args[0], api.ServiceOperationRequest{
			Operation: "stop",
		})
		if err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}

		fmt.Println(tui.Success.Render("Service stop initiated"))
		return nil
	},
}

func init() {
	stopCmd.Flags().StringVar(&stopServerFlag, "server", "", "Server ID")
}
