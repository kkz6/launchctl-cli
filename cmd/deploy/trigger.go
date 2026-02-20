package deploy

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/tui"
	deploytui "github.com/kkz6/launchctl/internal/tui/deploy"
	"github.com/spf13/cobra"
)

var triggerServerFlag string

var triggerCmd = &cobra.Command{
	Use:   "trigger <site-id>",
	Short: "Trigger a deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if triggerServerFlag == "" {
			return fmt.Errorf("--server flag is required")
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)
		siteID := args[0]

		site, err := client.GetSite(triggerServerFlag, siteID)
		if err != nil {
			return fmt.Errorf("failed to get site: %w", err)
		}

		deployment, err := client.DeploySite(triggerServerFlag, siteID)
		if err != nil {
			return fmt.Errorf("failed to trigger deployment: %w", err)
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Deployment %s triggered", deployment.ID)))

		jwt, err := client.ExchangeToken()
		if err != nil {
			fmt.Println(tui.Warning.Render("Could not authenticate for live logs, deployment is running in the background"))
			return nil
		}

		ws, err := api.NewWSClient(cfg, jwt)
		if err != nil {
			fmt.Println(tui.Warning.Render("Could not connect to live logs, deployment is running in the background"))
			return nil
		}
		defer ws.Close()

		channel := fmt.Sprintf("team:%s", cfg.TeamID)
		if err := ws.Subscribe(channel); err != nil {
			fmt.Println(tui.Warning.Render("Could not subscribe to events"))
			return nil
		}

		model := deploytui.NewModel(deploytui.Opts{
			SiteName:     site.Address,
			ServerID:     triggerServerFlag,
			SiteID:       siteID,
			DeploymentID: deployment.ID,
			Client:       client,
			JWT:          jwt,
			TeamID:       cfg.TeamID,
			WS:           ws,
		})
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	triggerCmd.Flags().StringVar(&triggerServerFlag, "server", "", "Server ID (required)")
}
