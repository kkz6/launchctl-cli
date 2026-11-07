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

func newTriggerCommand() *cobra.Command {
	var server string
	var wait bool
	var timeout int
	cmd := &cobra.Command{
		Use:          "trigger [site-id]",
		Short:        "Trigger a deployment",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateWaitOptions(wait, timeout); err != nil {
				return err
			}
			serverID, err := resolve.ServerID(server)
			if err != nil {
				return err
			}
			siteArg := ""
			if len(args) > 0 {
				siteArg = args[0]
			}
			siteID, err := resolve.SiteID(siteArg)
			if err != nil {
				return fmt.Errorf("site ID is required (or run `lctl init` to set a default)")
			}

			cfg := appstate.GetConfig()
			client := api.NewClient(cfg)
			deployment, err := client.DeploySiteContext(cmd.Context(), serverID, siteID)
			if err != nil {
				return fmt.Errorf("failed to trigger deployment: %w", err)
			}
			if wait {
				return reportDeploymentWait(cmd, client, serverID, siteID, deployment.ID, time.Duration(timeout)*time.Second)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, deployment)
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("Deployment %s triggered", deployment.ID)))
			if !interactive(cmd) {
				return nil
			}

			siteName := siteID
			if site, err := client.GetSite(serverID, siteID); err == nil {
				siteName = site.Address
			}
			jwt, err := client.ExchangeToken()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), tui.Warning.Render("Could not authenticate for live logs; deployment is running in the background"))
				return nil
			}
			ws, err := api.NewWSClient(cfg, jwt)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), tui.Warning.Render("Could not connect to live logs; deployment is running in the background"))
				return nil
			}
			defer ws.Close()
			if err := ws.Subscribe(fmt.Sprintf("team.%s", cfg.TeamID)); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), tui.Warning.Render("Could not subscribe to events; deployment is running in the background"))
				return nil
			}
			model := deploytui.NewModel(deploytui.Opts{
				Context:  cmd.Context(),
				SiteName: siteName, ServerID: serverID, SiteID: siteID,
				DeploymentID: deployment.ID, Client: client, JWT: jwt,
				TeamID: cfg.TeamID, APIURL: client.BaseURL(), WS: ws,
			})
			defer model.Close()
			final, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(cmd.Context())).Run()
			if err != nil {
				return err
			}
			if result, ok := final.(deploytui.Model); ok {
				result.Close()
				return result.Err()
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "Server ID (or project default)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for deployment to complete")
	cmd.Flags().IntVar(&timeout, "timeout", 300, "Timeout in seconds when using --wait")
	return cmd
}
