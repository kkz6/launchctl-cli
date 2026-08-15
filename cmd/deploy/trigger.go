package deploy

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	deploytui "github.com/kkz6/launchctl/internal/tui/deploy"
	"github.com/spf13/cobra"
)

var (
	triggerServerFlag  string
	triggerWaitFlag    bool
	triggerTimeoutFlag int
)

var triggerCmd = &cobra.Command{
	Use:   "trigger <site-id>",
	Short: "Trigger a deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(triggerServerFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		cfg.ApplyEnvOverrides()
		client := api.NewClient(cfg)
		siteID := args[0]

		ciMode, _ := cmd.Flags().GetBool("ci")

		site, err := client.GetSite(serverID, siteID)
		if err != nil {
			return fmt.Errorf("failed to get site: %w", err)
		}

		deployment, err := client.DeploySite(serverID, siteID)
		if err != nil {
			return fmt.Errorf("failed to trigger deployment: %w", err)
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Deployment %s triggered", deployment.ID)))

		if ciMode && triggerWaitFlag {
			return waitForDeployment(client, serverID, siteID, deployment.ID, time.Duration(triggerTimeoutFlag)*time.Second)
		}

		if ciMode {
			return nil
		}

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

		channel := fmt.Sprintf("team.%s", cfg.TeamID)
		if err := ws.Subscribe(channel); err != nil {
			fmt.Println(tui.Warning.Render("Could not subscribe to events"))
			return nil
		}

		model := deploytui.NewModel(deploytui.Opts{
			SiteName:     site.Address,
			ServerID:     serverID,
			SiteID:       siteID,
			DeploymentID: deployment.ID,
			Client:       client,
			JWT:          jwt,
			TeamID:       cfg.TeamID,
			APIURL:       client.BaseURL(),
			WS:           ws,
		})
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}

func waitForDeployment(client *api.Client, serverID, siteID, deploymentID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		d, err := client.GetDeployment(serverID, siteID, deploymentID)
		if err != nil {
			return fmt.Errorf("failed to check deployment status: %w", err)
		}

		switch d.Status {
		case "finished":
			fmt.Println(tui.Success.Render("Deployment completed successfully"))
			return nil
		case "failed":
			return fmt.Errorf("deployment failed")
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("deployment timed out after %s (status: %s)", timeout, d.Status)
		}

		<-ticker.C
	}
}

func init() {
	triggerCmd.Flags().StringVar(&triggerServerFlag, "server", "", "Server ID")
	triggerCmd.Flags().BoolVar(&triggerWaitFlag, "wait", false, "Wait for deployment to complete (CI/CD mode)")
	triggerCmd.Flags().IntVar(&triggerTimeoutFlag, "timeout", 300, "Timeout in seconds when using --wait")
}
