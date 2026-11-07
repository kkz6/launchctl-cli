package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	deploytui "github.com/kkz6/launchctl/internal/tui/deploy"
	"github.com/spf13/cobra"
)

var (
	logsServerFlag string
	logsFollowFlag bool
)

var logsCmd = &cobra.Command{
	SilenceUsage: true,
	Use:          "logs <site-id> [deployment-id]",
	Short:        "View deployment logs",
	Long:         "View output logs for the latest or a specific deployment. Use --follow to stream logs in real-time.",
	Args:         cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(logsServerFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)
		siteID := args[0]

		site, err := client.GetSite(serverID, siteID)
		if err != nil {
			return fmt.Errorf("failed to get site: %w", err)
		}

		var deployment *api.DeploymentResponse

		if len(args) == 2 {
			deployment, err = client.GetDeployment(serverID, siteID, args[1])
			if err != nil {
				return fmt.Errorf("failed to get deployment: %w", err)
			}
		} else {
			deployments, err := client.ListDeployments(serverID, siteID)
			if err != nil {
				return fmt.Errorf("failed to list deployments: %w", err)
			}

			if len(deployments) == 0 {
				return fmt.Errorf("no deployments found for site %s", site.Address)
			}

			deployment = &deployments[0]
		}

		isActive := deploymentLogsActive(deployment.Status)

		if logsFollowFlag {
			if !interactive(cmd) || !isActive {
				return followStoredLogs(cmd, client, serverID, siteID, deployment)
			}
			jwt, err := client.ExchangeToken()
			if err != nil {
				return fmt.Errorf("failed to authenticate for live logs: %w", err)
			}

			return streamLiveLogs(cmd.Context(), client, cfg, jwt, serverID, site, deployment)
		}

		return showStoredLogs(cmd, client, serverID, deployment)
	},
}

func streamLiveLogs(ctx context.Context, client *api.Client, cfg *config.Config, token, serverID string, site *api.SiteResponse, deployment *api.DeploymentResponse) error {
	ws, err := api.NewWSClient(cfg, token)
	if err != nil {
		return fmt.Errorf("failed to connect to live logs: %w", err)
	}
	defer ws.Close()

	channel := fmt.Sprintf("team.%s", cfg.TeamID)
	if err := ws.Subscribe(channel); err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	model := deploytui.NewModel(deploytui.Opts{
		Context:      ctx,
		SiteName:     site.Address,
		ServerID:     serverID,
		SiteID:       site.ID,
		DeploymentID: deployment.ID,
		Client:       client,
		JWT:          token,
		TeamID:       cfg.TeamID,
		APIURL:       client.BaseURL(),
		WS:           ws,
	})
	defer model.Close()
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	final, err := p.Run()
	if err != nil {
		return err
	}
	if model, ok := final.(deploytui.Model); ok {
		model.Close()
		return model.Err()
	}

	return nil
}

type deploymentLogsSnapshot struct {
	Deployment *api.DeploymentResponse `json:"deployment"`
	Task       *api.TaskResponse       `json:"task,omitempty"`
}

func deploymentLogsActive(status string) bool {
	return status == "pending" || status == "queued" || status == "installing" || status == "deploying"
}

func fetchDeploymentLogs(ctx context.Context, client *api.Client, serverID string, deployment *api.DeploymentResponse) (deploymentLogsSnapshot, error) {
	snapshot := deploymentLogsSnapshot{Deployment: deployment}
	if deployment.TaskID == nil || *deployment.TaskID == "" {
		return snapshot, nil
	}
	task, err := client.GetTaskContext(ctx, serverID, *deployment.TaskID)
	if err != nil {
		return snapshot, fmt.Errorf("failed to fetch deployment task %s: %w", *deployment.TaskID, err)
	}
	if task.ID != *deployment.TaskID || task.ServerID != serverID {
		return snapshot, fmt.Errorf("deployment task response does not match the requested task and server")
	}
	snapshot.Task = task
	return snapshot, nil
}

func (s deploymentLogsSnapshot) output() string {
	if s.Task == nil || s.Task.Output == nil {
		return ""
	}
	return *s.Task.Output
}

func showStoredLogs(cmd *cobra.Command, client *api.Client, serverID string, deployment *api.DeploymentResponse) error {
	snapshot, err := fetchDeploymentLogs(cmd.Context(), client, serverID, deployment)
	if err != nil {
		return err
	}
	if jsonEnabled(cmd) {
		return writeJSON(cmd, snapshot)
	}
	taskOutput := snapshot.output()

	if taskOutput == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "No output available for this deployment.")
		return nil
	}
	if !interactive(cmd) {
		_, err := fmt.Fprint(cmd.OutOrStdout(), taskOutput)
		return err
	}

	commit := deployment.ShortGitHash
	if deployment.CommitData != nil {
		commit += " " + deployment.CommitData.Message
	}

	fmt.Fprintln(cmd.OutOrStdout(), tui.Title.Render(fmt.Sprintf(" Deployment %s ", deployment.ID)))
	fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Status: ")+tui.Value.Render(deployment.Status))
	fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Commit: ")+tui.Value.Render(commit))
	fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Created: ")+tui.Value.Render(deployment.CreatedAt))
	fmt.Fprintln(cmd.OutOrStdout())

	lines := strings.Split(taskOutput, "\n")
	for _, line := range lines {
		fmt.Fprintln(cmd.OutOrStdout(), line)
	}

	return nil
}

// Noninteractive follow emits appended output, or changed NDJSON snapshots,
// while REST remains the source of truth for deployment completion.
func followStoredLogs(cmd *cobra.Command, client *api.Client, serverID, siteID string, deployment *api.DeploymentResponse) error {
	return followStoredLogsInterval(cmd, client, serverID, siteID, deployment, 2*time.Second)
}

func followStoredLogsInterval(cmd *cobra.Command, client *api.Client, serverID, siteID string, deployment *api.DeploymentResponse, interval time.Duration) error {
	var previousOutput, previousJSON string
	for {
		snapshot, err := fetchDeploymentLogs(cmd.Context(), client, serverID, deployment)
		if err != nil {
			return err
		}
		if jsonEnabled(cmd) {
			data, err := json.Marshal(snapshot)
			if err != nil {
				return err
			}
			if string(data) != previousJSON {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
					return err
				}
				previousJSON = string(data)
			}
		} else {
			output := snapshot.output()
			added := output
			if strings.HasPrefix(output, previousOutput) {
				added = strings.TrimPrefix(output, previousOutput)
			}
			if _, err := fmt.Fprint(cmd.OutOrStdout(), added); err != nil {
				return err
			}
			previousOutput = output
		}
		switch deployment.Status {
		case "finished":
			return nil
		case "failed", "timeout", "cancelled":
			return fmt.Errorf("deployment %s %s", deployment.ID, deployment.Status)
		}
		if !deploymentLogsActive(deployment.Status) {
			return fmt.Errorf("deployment %s has unknown status %q", deployment.ID, deployment.Status)
		}
		timer := time.NewTimer(interval)
		select {
		case <-cmd.Context().Done():
			timer.Stop()
			return cmd.Context().Err()
		case <-timer.C:
		}
		deployment, err = client.GetDeploymentContext(cmd.Context(), serverID, siteID, deployment.ID)
		if err != nil {
			return fmt.Errorf("failed to refresh deployment: %w", err)
		}
	}
}

func init() {
	logsCmd.Flags().StringVar(&logsServerFlag, "server", "", "Server ID (required)")
	logsCmd.Flags().BoolVarP(&logsFollowFlag, "follow", "f", false, "Stream logs in real-time for active deployments")
}
