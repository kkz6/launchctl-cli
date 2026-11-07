package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

func validateWaitOptions(wait bool, timeout int) error {
	if wait && (timeout <= 0 || int64(timeout) > int64((1<<63-1)/time.Second)) {
		return fmt.Errorf("--timeout must be a positive number of seconds within the supported duration range")
	}
	return nil
}

func reportDeploymentWait(cmd *cobra.Command, client *api.Client, serverID, siteID, deploymentID string, timeout time.Duration) error {
	deployment, waitErr := waitForDeployment(cmd.Context(), client, serverID, siteID, deploymentID, timeout)
	if jsonEnabled(cmd) && deployment != nil {
		if err := writeJSON(cmd, deployment); err != nil {
			return err
		}
	}
	if waitErr != nil {
		return waitErr
	}
	if !jsonEnabled(cmd) {
		fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("Deployment %s completed successfully", deployment.ID)))
	}
	return nil
}

func waitForDeployment(ctx context.Context, client *api.Client, serverID, siteID, deploymentID string, timeout time.Duration) (*api.DeploymentResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	var last *api.DeploymentResponse
	for {
		deployment, err := client.GetDeploymentContext(ctx, serverID, siteID, deploymentID)
		if err != nil {
			return last, fmt.Errorf("failed to check deployment %s: %w", deploymentID, err)
		}
		last = deployment
		switch deployment.Status {
		case "finished":
			return deployment, nil
		case "failed", "timeout", "cancelled":
			return deployment, fmt.Errorf("deployment %s ended with status %s", deploymentID, deployment.Status)
		}
		select {
		case <-ctx.Done():
			return last, fmt.Errorf("waiting for deployment %s stopped (status: %s): %w", deploymentID, last.Status, ctx.Err())
		case <-ticker.C:
		}
	}
}
