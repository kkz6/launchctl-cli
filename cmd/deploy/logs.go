package deploy

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/tui"
	deploytui "github.com/kkz6/launchctl/internal/tui/deploy"
	"github.com/spf13/cobra"
)

var (
	logsServerFlag string
	logsFollowFlag bool
)

var logsCmd = &cobra.Command{
	Use:   "logs <site-id> [deployment-id]",
	Short: "View deployment logs",
	Long:  "View output logs for the latest or a specific deployment. Use --follow to stream logs in real-time.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if logsServerFlag == "" {
			return fmt.Errorf("--server flag is required")
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)
		siteID := args[0]

		site, err := client.GetSite(logsServerFlag, siteID)
		if err != nil {
			return fmt.Errorf("failed to get site: %w", err)
		}

		var deployment *api.DeploymentResponse

		if len(args) == 2 {
			deployment, err = client.GetDeployment(logsServerFlag, siteID, args[1])
			if err != nil {
				return fmt.Errorf("failed to get deployment: %w", err)
			}
		} else {
			deployments, err := client.ListDeployments(logsServerFlag, siteID)
			if err != nil {
				return fmt.Errorf("failed to list deployments: %w", err)
			}

			if len(deployments) == 0 {
				return fmt.Errorf("no deployments found for site %s", site.Address)
			}

			deployment = &deployments[0]
		}

		isActive := deployment.Status == "deploying" || deployment.Status == "pending"

		if logsFollowFlag && isActive {
			jwt, err := client.ExchangeToken()
			if err != nil {
				return fmt.Errorf("failed to authenticate for live logs: %w", err)
			}

			return streamLiveLogs(cfg, jwt, site, deployment)
		}

		return showStoredLogs(client, logsServerFlag, deployment)
	},
}

func streamLiveLogs(cfg *config.Config, token string, site *api.SiteResponse, deployment *api.DeploymentResponse) error {
	ws, err := api.NewWSClient(cfg, token)
	if err != nil {
		return fmt.Errorf("failed to connect to live logs: %w", err)
	}
	defer ws.Close()

	channel := fmt.Sprintf("team:%s", cfg.TeamID)
	if err := ws.Subscribe(channel); err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	model := deploytui.NewModel(site.Address, ws, channel)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func showStoredLogs(client *api.Client, serverID string, deployment *api.DeploymentResponse) error {
	if deployment.TaskID == nil {
		return fmt.Errorf("deployment has no associated task")
	}

	tasks, err := client.ListTasks(serverID)
	if err != nil {
		return fmt.Errorf("failed to fetch tasks: %w", err)
	}

	var taskOutput string
	for _, t := range tasks {
		if t.ID == *deployment.TaskID {
			if t.Output != nil {
				taskOutput = *t.Output
			}
			break
		}
	}

	if taskOutput == "" {
		fmt.Println(tui.Dim.Render("No output available for this deployment."))
		return nil
	}

	commit := deployment.ShortGitHash
	if deployment.CommitData != nil {
		commit += " " + deployment.CommitData.Message
	}

	fmt.Println(tui.Title.Render(fmt.Sprintf(" Deployment %s ", deployment.ID[:8])))
	fmt.Println(tui.Label.Render("Status: ") + tui.Value.Render(deployment.Status))
	fmt.Println(tui.Label.Render("Commit: ") + tui.Value.Render(commit))
	fmt.Println(tui.Label.Render("Created: ") + tui.Value.Render(deployment.CreatedAt))
	fmt.Println()

	lines := strings.Split(taskOutput, "\n")
	for _, line := range lines {
		fmt.Println(line)
	}

	return nil
}

func init() {
	logsCmd.Flags().StringVar(&logsServerFlag, "server", "", "Server ID (required)")
	logsCmd.Flags().BoolVarP(&logsFollowFlag, "follow", "f", false, "Stream logs in real-time for active deployments")
}
