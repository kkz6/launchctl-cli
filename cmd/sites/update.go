package sites

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/commandutil"
	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	var server, phpVersion, webFolder, branch string
	var retention int
	var queue bool
	cmd := &cobra.Command{
		Use:          "update [site-id]",
		Short:        "Update site settings",
		Long:         "Update only the settings supplied as flags. PHP and web-folder changes may apply asynchronously. Site and server default to .launchctl.yml.",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := api.UpdateSiteRequest{}
			if cmd.Flags().Changed("php-version") {
				switch phpVersion {
				case "php80", "php81", "php82", "php83", "php84":
				default:
					return fmt.Errorf("--php-version must be php80, php81, php82, php83, or php84")
				}
				req.PHPVersion = &phpVersion
			}
			if cmd.Flags().Changed("web-folder") {
				req.WebFolder = &webFolder
			}
			if cmd.Flags().Changed("branch") {
				req.RepositoryBranch = &branch
			}
			if cmd.Flags().Changed("release-retention") {
				if retention < 1 || retention > 50 {
					return fmt.Errorf("--release-retention must be between 1 and 50")
				}
				req.DeploymentReleasesRetention = &retention
			}
			if cmd.Flags().Changed("queue-deployments") {
				req.QueueDeployments = &queue
			}
			if req == (api.UpdateSiteRequest{}) {
				return fmt.Errorf("provide at least one setting: --php-version, --web-folder, --branch, --release-retention, or --queue-deployments")
			}
			serverID, siteID, err := siteTarget(server, args)
			if err != nil {
				return err
			}
			result, err := api.NewClient(appstate.GetConfig()).UpdateSite(cmd.Context(), serverID, siteID, req)
			if err != nil {
				return fmt.Errorf("failed to update site: %w", err)
			}
			if commandutil.JSONEnabled(cmd) {
				return commandutil.WriteJSON(cmd, result)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Update requested for site %s.\n", siteID)
			return err
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "Server ID (or project default)")
	cmd.Flags().StringVar(&phpVersion, "php-version", "", "Active PHP version installed on the server: php80 through php84")
	cmd.Flags().StringVar(&webFolder, "web-folder", "", "Web folder (empty clears it)")
	cmd.Flags().StringVar(&branch, "branch", "", "Repository branch (empty clears it)")
	cmd.Flags().IntVar(&retention, "release-retention", 0, "Number of releases to retain (1-50)")
	cmd.Flags().BoolVar(&queue, "queue-deployments", false, "Queue deployments while another runs (use =false to disable)")
	return cmd
}
