package deploy

import (
	"fmt"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

func newRollbackCommand() *cobra.Command {
	var server, site string
	var yes, wait bool
	var timeout int
	cmd := &cobra.Command{
		Use:          "rollback <deployment-id>",
		Short:        "Roll back to a previous deployment",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateWaitOptions(wait, timeout); err != nil {
				return err
			}
			serverID, err := resolve.ServerID(server)
			if err != nil {
				return err
			}
			siteID, err := resolve.SiteID(site)
			if err != nil {
				return err
			}
			confirmed, err := confirmDestructive(cmd, yes, "Roll back to this deployment?", "This will deploy the selected previous version of the site.")
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Fprintln(cmd.OutOrStdout(), tui.Dim.Render("Cancelled"))
				return nil
			}

			client := api.NewClient(appstate.GetConfig())
			deployment, err := client.RollbackDeploymentContext(cmd.Context(), serverID, siteID, args[0])
			if err != nil {
				return fmt.Errorf("failed to roll back: %w", err)
			}
			if wait {
				return reportDeploymentWait(cmd, client, serverID, siteID, deployment.ID, time.Duration(timeout)*time.Second)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, deployment)
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("Rollback triggered: %s", deployment.ID)))
			return nil
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "Server ID (or project default)")
	cmd.Flags().StringVar(&site, "site", "", "Site ID (or project default)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm the rollback without prompting")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for rollback to complete")
	cmd.Flags().IntVar(&timeout, "timeout", 300, "Timeout in seconds when using --wait")
	return cmd
}
