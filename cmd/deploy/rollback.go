package deploy

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var rollbackServerFlag, rollbackSiteFlag string

var rollbackCmd = &cobra.Command{
	Use:   "rollback <deployment-id>",
	Short: "Rollback to a previous deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(rollbackServerFlag)
		if err != nil {
			return err
		}

		siteID, err := resolve.SiteID(rollbackSiteFlag)
		if err != nil {
			return err
		}

		var confirm bool
		huh.NewConfirm().
			Title("Rollback to this deployment?").
			Description("This will deploy the previous version of the site.").
			Value(&confirm).
			Run()

		if !confirm {
			fmt.Println(tui.Dim.Render("Cancelled"))
			return nil
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		d, err := client.RollbackDeployment(serverID, siteID, args[0])
		if err != nil {
			return fmt.Errorf("failed to rollback: %w", err)
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Rollback triggered: %s", d.ID)))
		return nil
	},
}

func init() {
	rollbackCmd.Flags().StringVar(&rollbackServerFlag, "server", "", "Server ID (required)")
	rollbackCmd.Flags().StringVar(&rollbackSiteFlag, "site", "", "Site ID (required)")
}
