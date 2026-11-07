package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/redact"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

const maxComposeFileBytes = 128 << 10

type composesOptions struct {
	server  string
	project string
}

func newComposesCommand() *cobra.Command {
	opts := &composesOptions{}
	cmd := &cobra.Command{
		Use: "composes", Aliases: []string{"compose"}, Short: "Manage Docker Compose stacks", SilenceUsage: true,
	}
	cmd.PersistentFlags().StringVar(&opts.server, "server", "", "Docker server ID (or project default)")
	cmd.PersistentFlags().StringVar(&opts.project, "project", "", "Docker project ID (or project default)")
	cmd.AddCommand(newComposesListCommand(opts), newComposesShowCommand(opts), newComposesCreateCommand(opts), newComposesUpdateCommand(opts))
	cmd.AddCommand(newComposesDeployCommand(opts), newComposesReloadCommand(opts), newComposesDeleteCommand(opts), newComposesDeploymentsCommand(opts))
	for _, child := range cmd.Commands() {
		child.SilenceUsage = true
	}
	return cmd
}

func dockerComposeContext(cmd *cobra.Command, opts *composesOptions) (*api.Client, string, string, error) {
	projectID, err := resolve.ProjectID(strings.TrimSpace(opts.project))
	if err != nil {
		return nil, "", "", err
	}
	serverID, err := resolve.ServerID(strings.TrimSpace(opts.server))
	if err != nil {
		return nil, "", "", err
	}
	client := api.NewClient(appstate.GetConfig())
	server, err := client.GetServerContext(cmd.Context(), serverID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get server: %w", err)
	}
	if server.Type != "docker" {
		return nil, "", "", fmt.Errorf("server %q is not a Docker application server", server.Name)
	}
	return client, serverID, projectID, nil
}

func composeIDArg(cmd *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		return err
	}
	if strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("compose ID is required")
	}
	return nil
}

func newComposesListCommand(opts *composesOptions) *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List stacks in a Docker project", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, serverID, projectID, err := dockerComposeContext(cmd, opts)
			if err != nil {
				return err
			}
			stacks, err := client.ListDockerComposesContext(cmd.Context(), serverID, projectID)
			if err != nil {
				return fmt.Errorf("failed to list Compose stacks: %w", err)
			}
			rows := make([][]string, 0, len(stacks))
			for i := range stacks {
				stacks[i] = sanitizedDockerCompose(stacks[i], false)
				rows = append(rows, []string{stacks[i].ID, stacks[i].Name, stacks[i].ComposeSourceType, stacks[i].Status, formatTime(stacks[i].LastDeployedAt)})
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, stacks)
			}
			return renderComposeTable(cmd, []string{"ID", "Name", "Source", "Status", "Last Deployed"}, rows)
		},
	}
}

func newComposesShowCommand(opts *composesOptions) *cobra.Command {
	var showConfig bool
	cmd := &cobra.Command{
		Use: "show <compose-id>", Short: "Show a Compose stack", Args: composeIDArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showConfig && !jsonEnabled(cmd) {
				return fmt.Errorf("--show-config requires --json")
			}
			client, serverID, projectID, err := dockerComposeContext(cmd, opts)
			if err != nil {
				return err
			}
			stack, err := client.GetDockerComposeContext(cmd.Context(), serverID, projectID, strings.TrimSpace(args[0]))
			if err != nil {
				return fmt.Errorf("failed to get Compose stack: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, sanitizedDockerCompose(*stack, showConfig))
			}
			return renderComposeTable(cmd, []string{"Field", "Value"}, [][]string{
				{"ID", stack.ID}, {"Name", stack.Name}, {"Project", stack.ProjectID}, {"Status", stack.Status},
				{"Source", stack.ComposeSourceType}, {"Build Location", stack.BuildLocation},
				{"Compose File", valueOrEmpty(stack.ComposeFilePath)}, {"Last Deployed", formatTime(stack.LastDeployedAt)},
			})
		},
	}
	cmd.Flags().BoolVar(&showConfig, "show-config", false, "Include YAML, environment, and command contents in --json output (may contain secrets)")
	return cmd
}

func newComposesCreateCommand(opts *composesOptions) *cobra.Command {
	var name, file string
	cmd := &cobra.Command{
		Use: "create", Short: "Register a stack from a local Compose YAML file", Args: cobra.NoArgs,
		Long: "Register a stack from a local Compose YAML file without deploying it. Use the deploy command to start the stack.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			contents, err := readComposeFile(file, false)
			if err != nil {
				return fmt.Errorf("--file: %w", err)
			}
			client, serverID, projectID, err := dockerComposeContext(cmd, opts)
			if err != nil {
				return err
			}
			stack, err := client.CreateDockerComposeContext(cmd.Context(), serverID, projectID, api.CreateDockerComposeRequest{
				Name: name, ComposeSourceType: "raw_yaml", RawYAML: &api.DockerComposeRawYAML{Contents: contents},
			})
			if err != nil {
				return fmt.Errorf("failed to create Compose stack: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, sanitizedDockerCompose(*stack, false))
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Compose stack %s created. Run deploy to start it.\n", stack.ID)
			return err
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Stack name (required)")
	cmd.Flags().StringVar(&file, "file", "", "Local Compose YAML file, up to 128 KiB (required)")
	return cmd
}

func newComposesUpdateCommand(opts *composesOptions) *cobra.Command {
	var name, envFile, runCommand string
	cmd := &cobra.Command{
		Use: "update <compose-id>", Short: "Update a stack's name or saved runtime settings", Args: composeIDArg,
		Long: "Update the name, saved environment, or command override. Environment changes apply on the next deploy or reload; command changes require a deploy. The API does not support changing the Compose YAML or source.",
		RunE: func(cmd *cobra.Command, args []string) error {
			req := api.UpdateDockerComposeRequest{}
			if cmd.Flags().Changed("name") {
				name = strings.TrimSpace(name)
				if name == "" {
					return fmt.Errorf("--name cannot be empty")
				}
				req.Name = &name
			}
			if cmd.Flags().Changed("env-file") {
				contents, err := readComposeFile(envFile, true)
				if err != nil {
					return fmt.Errorf("--env-file: %w", err)
				}
				req.EnvFile = &contents
			}
			if cmd.Flags().Changed("run-command") {
				if len(runCommand) > 8192 {
					return fmt.Errorf("--run-command exceeds 8 KiB limit")
				}
				req.RunCommand = &runCommand
			}
			if req.Name == nil && req.EnvFile == nil && req.RunCommand == nil {
				return fmt.Errorf("provide --name, --env-file, or --run-command")
			}
			client, serverID, projectID, err := dockerComposeContext(cmd, opts)
			if err != nil {
				return err
			}
			stack, err := client.UpdateDockerComposeContext(cmd.Context(), serverID, projectID, strings.TrimSpace(args[0]), req)
			if err != nil {
				return fmt.Errorf("failed to update Compose stack: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, sanitizedDockerCompose(*stack, false))
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Compose stack %s updated. Environment changes apply on deploy or reload; command changes require deploy.\n", stack.ID)
			return err
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New stack name")
	cmd.Flags().StringVar(&envFile, "env-file", "", "Local environment file, up to 128 KiB; an empty file clears saved variables")
	cmd.Flags().StringVar(&runCommand, "run-command", "", "Docker command suffix override; an empty string restores the default")
	return cmd
}

func newComposesDeployCommand(opts *composesOptions) *cobra.Command {
	var wait bool
	var timeout int
	cmd := &cobra.Command{
		Use: "deploy <compose-id>", Short: "Deploy or rebuild a Compose stack", Args: composeIDArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateDockerWaitOptions(wait, timeout); err != nil {
				return err
			}
			client, serverID, projectID, err := dockerComposeContext(cmd, opts)
			if err != nil {
				return err
			}
			composeID := strings.TrimSpace(args[0])
			deployment, err := client.DeployDockerComposeContext(cmd.Context(), serverID, projectID, composeID)
			if err != nil {
				return fmt.Errorf("failed to deploy Compose stack: %w", err)
			}
			if deployment.ID == "" {
				return fmt.Errorf("deploy response did not include a deployment ID; check deployment history before retrying")
			}
			var waitErr error
			if wait {
				ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(timeout)*time.Second)
				defer cancel()
				final, err := waitForComposeDeployment(ctx, client, serverID, projectID, composeID, deployment.ID)
				if final != nil {
					deployment = final
				}
				waitErr = err
			}
			if jsonEnabled(cmd) {
				if err := writeJSON(cmd, deployment); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Compose deployment %s: %s\n", deployment.ID, deployment.Status)
			}
			return waitErr
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for this deployment's terminal result")
	cmd.Flags().IntVar(&timeout, "timeout", 600, "Wait timeout in seconds")
	return cmd
}

func newComposesReloadCommand(opts *composesOptions) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use: "reload <compose-id>", Short: "Recreate a stack with saved settings without rebuilding", Args: composeIDArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			confirmed, err := confirmDestructive(cmd, yes, "Reload this Compose stack?", "Running services will be recreated with the saved environment and existing images.")
			if err != nil || !confirmed {
				return err
			}
			client, serverID, projectID, err := dockerComposeContext(cmd, opts)
			if err != nil {
				return err
			}
			composeID := strings.TrimSpace(args[0])
			if err := client.ReloadDockerComposeContext(cmd.Context(), serverID, projectID, composeID); err != nil {
				return fmt.Errorf("failed to reload Compose stack: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, map[string]string{"compose_id": composeID, "action": "reload", "status": "queued"})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Reload queued for Compose stack %s.\n", composeID)
			return err
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm service recreation without prompting")
	return cmd
}

func newComposesDeleteCommand(opts *composesOptions) *cobra.Command {
	var yes, removeVolumes bool
	cmd := &cobra.Command{
		Use: "delete <compose-id>", Short: "Delete a Compose stack", Args: composeIDArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			volumeNotice := "Named volumes will be preserved."
			if removeVolumes {
				volumeNotice = "Named volumes will also be permanently removed."
			}
			confirmed, err := confirmDestructive(cmd, yes, "Delete this Compose stack?", volumeNotice)
			if err != nil || !confirmed {
				return err
			}
			client, serverID, projectID, err := dockerComposeContext(cmd, opts)
			if err != nil {
				return err
			}
			composeID := strings.TrimSpace(args[0])
			if err := client.DeleteDockerComposeContext(cmd.Context(), serverID, projectID, composeID, removeVolumes); err != nil {
				return fmt.Errorf("failed to delete Compose stack: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, map[string]any{"compose_id": composeID, "remove_volumes": removeVolumes, "status": "deleting"})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Deletion queued for Compose stack %s. %s\n", composeID, volumeNotice)
			return err
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm deletion without prompting")
	cmd.Flags().BoolVar(&removeVolumes, "remove-volumes", false, "Permanently remove the stack's named volumes")
	return cmd
}

func newComposesDeploymentsCommand(opts *composesOptions) *cobra.Command {
	return &cobra.Command{
		Use: "deployments <compose-id>", Short: "List Compose deployment history", Args: composeIDArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, serverID, projectID, err := dockerComposeContext(cmd, opts)
			if err != nil {
				return err
			}
			deployments, err := client.ListDockerComposeDeploymentsContext(cmd.Context(), serverID, projectID, strings.TrimSpace(args[0]))
			if err != nil {
				return fmt.Errorf("failed to list Compose deployments: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, deployments)
			}
			rows := make([][]string, 0, len(deployments))
			for _, deployment := range deployments {
				rows = append(rows, []string{deployment.ID, deployment.Status, deployment.TriggerSource, valueOrEmpty(deployment.TaskID), formatTime(deployment.CreatedAt), formatTime(deployment.FinishedAt)})
			}
			return renderComposeTable(cmd, []string{"ID", "Status", "Trigger", "Task", "Created", "Finished"}, rows)
		},
	}
}

func waitForComposeDeployment(ctx context.Context, client *api.Client, serverID, projectID, composeID, deploymentID string) (*api.DockerDeploymentResponse, error) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	var last *api.DockerDeploymentResponse
	lastStatus := "pending"
	for {
		deployments, err := client.ListDockerComposeDeploymentsContext(ctx, serverID, projectID, composeID)
		if err != nil {
			if ctx.Err() != nil {
				return last, dockerWaitContextError(ctx, "Compose deployment "+deploymentID, lastStatus)
			}
			return last, fmt.Errorf("failed to check Compose deployment %s: %w", deploymentID, err)
		}
		for i := range deployments {
			deployment := &deployments[i]
			if deployment.ID != deploymentID {
				continue
			}
			if deployment.ServerID != serverID || deployment.TargetType != "compose" || deployment.TargetID != composeID {
				return last, fmt.Errorf("deployment %s does not match the requested Compose stack", deploymentID)
			}
			last, lastStatus = deployment, deployment.Status
			if terminal, err := dockerDeploymentOutcome(deployment); terminal {
				return deployment, err
			}
			break
		}
		select {
		case <-ctx.Done():
			return last, dockerWaitContextError(ctx, "Compose deployment "+deploymentID, lastStatus)
		case <-ticker.C:
		}
	}
}

func readComposeFile(path string, allowEmpty bool) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("a local file path is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxComposeFileBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxComposeFileBytes {
		return "", fmt.Errorf("file exceeds 128 KiB limit")
	}
	if !allowEmpty && strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("Compose YAML cannot be empty")
	}
	return string(data), nil
}

func sanitizedDockerCompose(stack api.DockerComposeResponse, showConfig bool) api.DockerComposeResponse {
	stack.SourceConfig = sanitizedDockerApplication(api.DockerApplicationResponse{SourceConfig: stack.SourceConfig}).SourceConfig
	stack.RegistryCredentials = append([]api.DockerRegistryCredentialSummary(nil), stack.RegistryCredentials...)
	for i := range stack.RegistryCredentials {
		if stack.RegistryCredentials[i].RegistryURL != nil {
			value := redact.URL(*stack.RegistryCredentials[i].RegistryURL)
			stack.RegistryCredentials[i].RegistryURL = &value
		}
	}
	if !showConfig {
		redacted := "REDACTED"
		for _, field := range []**string{&stack.RawYAML, &stack.EnvFile, &stack.RunCommand} {
			if *field != nil {
				*field = &redacted
			}
		}
	}
	return stack
}

func renderComposeTable(cmd *cobra.Command, headers []string, rows [][]string) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	return w.Flush()
}
