package sites

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/commandutil"
	"github.com/spf13/cobra"
)

func newAutoDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-deploy",
		Short: "Enable or disable automatic deployments",
	}
	cmd.AddCommand(newAutoDeployActionCommand(true), newAutoDeployActionCommand(false))
	return cmd
}

func newAutoDeployActionCommand(enabled bool) *cobra.Command {
	action := "disable"
	if enabled {
		action = "enable"
	}
	var server string
	cmd := &cobra.Command{
		Use:          action + " [site-id]",
		Short:        action + " automatic deployments for a site",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID, siteID, err := siteTarget(server, args)
			if err != nil {
				return err
			}
			if err := api.NewClient(appstate.GetConfig()).SetSiteAutoDeployment(cmd.Context(), serverID, siteID, enabled); err != nil {
				return fmt.Errorf("failed to %s automatic deployments: %w", action, err)
			}
			if commandutil.JSONEnabled(cmd) {
				return commandutil.WriteJSON(cmd, map[string]any{"server_id": serverID, "site_id": siteID, "auto_deployment": enabled})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Automatic deployments %sd for site %s.\n", action, siteID)
			return err
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "Server ID (or project default)")
	return cmd
}
