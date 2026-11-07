package deploy

import (
	"fmt"
	"strings"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

func newCancelQueuedCommand() *cobra.Command {
	var serverFlag string
	var yes bool
	cmd := &cobra.Command{
		Use:          "cancel-queued [site-id]",
		Short:        "Cancel queued deployments for a site",
		Long:         "Cancel all queued deployments for a site. Deployments already running are unaffected. Site and server default to .launchctl.yml.",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID, err := resolve.ServerID(strings.TrimSpace(serverFlag))
			if err != nil {
				return err
			}
			siteArg := ""
			if len(args) > 0 {
				siteArg = strings.TrimSpace(args[0])
			}
			siteID, err := resolve.SiteID(siteArg)
			if err != nil {
				return fmt.Errorf("site ID is required (or run `lctl init` to set a default)")
			}
			confirmed, err := confirmDestructive(cmd, yes, "Cancel queued deployments?", fmt.Sprintf("Cancel all queued deployments for site %s on server %s.", siteID, serverID))
			if err != nil {
				return err
			}
			if !confirmed {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
				return err
			}

			client := api.NewClient(appstate.GetConfig())
			result, err := client.CancelQueuedDeployments(cmd.Context(), serverID, siteID)
			if err != nil {
				return fmt.Errorf("failed to cancel queued deployments: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, result)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Cancelled %d queued deployment(s).\n", result.Cancelled)
			return err
		},
	}
	cmd.Flags().StringVar(&serverFlag, "server", "", "Server ID (default: project config)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm cancellation without prompting")
	return cmd
}
