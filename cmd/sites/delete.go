package sites

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/commandutil"
	"github.com/spf13/cobra"
)

func newDeleteCommand() *cobra.Command {
	var server string
	var yes bool
	cmd := &cobra.Command{
		Use:          "delete [site-id]",
		Short:        "Request deletion of a site",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID, siteID, err := siteTarget(server, args)
			if err != nil {
				return err
			}
			confirmed, err := commandutil.Confirm(cmd, yes, "Delete this site?", fmt.Sprintf("Request deletion of site %s on server %s. This cannot be undone.", siteID, serverID))
			if err != nil {
				return err
			}
			if !confirmed {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
				return err
			}
			if err := api.NewClient(appstate.GetConfig()).DeleteSite(cmd.Context(), serverID, siteID); err != nil {
				return fmt.Errorf("failed to request site deletion: %w", err)
			}
			if commandutil.JSONEnabled(cmd) {
				return commandutil.WriteJSON(cmd, map[string]any{"deletion_requested": true, "server_id": serverID, "site_id": siteID})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Deletion requested for site %s.\n", siteID)
			return err
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "Server ID (or project default)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm deletion without prompting")
	return cmd
}
