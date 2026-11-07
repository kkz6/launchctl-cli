package sites

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/commandutil"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

func newCreateCommand() *cobra.Command {
	var server, sourceControl, repositoryID string
	var req api.CreateSiteRequest
	cmd := &cobra.Command{
		Use:          "create",
		Short:        "Create a site and request its initial deployment",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			req.Address = strings.TrimSpace(req.Address)
			req.Type = strings.ToLower(strings.TrimSpace(req.Type))
			req.PHPVersion = strings.TrimSpace(req.PHPVersion)
			if req.Address == "" {
				return fmt.Errorf("--address is required")
			}
			switch req.Type {
			case "laravel", "wordpress", "static", "generic", "phpmyadmin":
			default:
				return fmt.Errorf("--type must be laravel, wordpress, static, generic, or phpmyadmin")
			}
			if req.PHPVersion == "" {
				return fmt.Errorf("--php-version is required (use an installed version such as php84)")
			}
			sourceControl, repositoryID = strings.TrimSpace(sourceControl), strings.TrimSpace(repositoryID)
			if (sourceControl == "") != (repositoryID == "") {
				return fmt.Errorf("--source-control and --repository-id must be provided together")
			}
			if req.Type == "wordpress" || req.Type == "phpmyadmin" {
				if sourceControl != "" || cmd.Flags().Changed("branch") || req.ZeroDowntime {
					return fmt.Errorf("%s does not support repository settings or zero-downtime deployment", req.Type)
				}
			}
			if sourceControl != "" {
				id, err := strconv.ParseUint(repositoryID, 10, 64)
				if err != nil || id == 0 {
					return fmt.Errorf("--repository-id must be a positive numeric repository ID")
				}
				req.SourceControlID, req.SourceControlRepositoriesID = &sourceControl, &repositoryID
			}
			serverID, err := resolve.ServerID(strings.TrimSpace(server))
			if err != nil {
				return err
			}
			result, err := api.NewClient(appstate.GetConfig()).CreateSiteContext(cmd.Context(), serverID, req)
			if err != nil {
				return fmt.Errorf("failed to create site: %w", err)
			}
			if commandutil.JSONEnabled(cmd) {
				return commandutil.WriteJSON(cmd, result)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Site %q created (ID: %s).\n", result.Address, result.ID)
			return err
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "Server ID (or project default)")
	cmd.Flags().StringVar(&req.Address, "address", "", "Site hostname (required)")
	cmd.Flags().StringVar(&req.Type, "type", "", "Site type: laravel, wordpress, static, generic, phpmyadmin (required)")
	cmd.Flags().StringVar(&req.PHPVersion, "php-version", "", "Active PHP version installed on the server, such as php84 (required)")
	cmd.Flags().StringVar(&req.WebFolder, "web-folder", "", "Web folder (default: backend default for the site type)")
	cmd.Flags().BoolVar(&req.ZeroDowntime, "zero-downtime", false, "Enable release-based zero-downtime deployments")
	cmd.Flags().StringVar(&sourceControl, "source-control", "", "Connected source-control account ID")
	cmd.Flags().StringVar(&repositoryID, "repository-id", "", "Numeric repository ID from the connected source-control account")
	cmd.Flags().StringVar(&req.RepositoryBranch, "branch", "", "Repository branch")
	return cmd
}
