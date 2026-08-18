package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/redact"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

const maxDockerfileBytes = 64 << 10

type applicationsOptions struct {
	server  string
	project string
}

type applicationCreateOptions struct {
	name               string
	source             string
	port               int
	image              string
	registryCredential string
	repo               string
	branch             string
	sourceControl      string
	buildType          string
	dockerfilePath     string
	buildLocation      string
	dockerfile         string
}

func newApplicationsCommand() *cobra.Command {
	opts := &applicationsOptions{}
	cmd := &cobra.Command{
		Use:     "applications",
		Aliases: []string{"application", "apps", "app"},
		Short:   "Manage Docker applications",
	}
	cmd.PersistentFlags().StringVar(&opts.server, "server", "", "Docker server ID (or project default)")
	cmd.PersistentFlags().StringVar(&opts.project, "project", "", "Docker project ID (or project default)")
	cmd.AddCommand(newApplicationsListCommand(opts))
	cmd.AddCommand(newApplicationsShowCommand(opts))
	cmd.AddCommand(newApplicationsCreateCommand(opts))
	cmd.AddCommand(newApplicationsUpdateCommand(opts))
	cmd.AddCommand(newApplicationsDeployCommand(opts))
	cmd.AddCommand(newApplicationActionCommand(opts, "reload"))
	cmd.AddCommand(newApplicationActionCommand(opts, "start"))
	cmd.AddCommand(newApplicationActionCommand(opts, "stop"))
	cmd.AddCommand(newApplicationsDeploymentsCommand(opts))
	cmd.AddCommand(newApplicationsDeleteCommand(opts))
	return cmd
}

func dockerApplicationContext(opts *applicationsOptions, args []string) (*api.Client, string, string, string, error) {
	applicationID, err := applicationIDFromArg(args)
	if err != nil {
		return nil, "", "", "", err
	}
	projectID, err := resolve.ProjectID(strings.TrimSpace(opts.project))
	if err != nil {
		return nil, "", "", "", err
	}
	client, serverID, err := dockerClient(opts.server)
	if err != nil {
		return nil, "", "", "", err
	}
	return client, serverID, projectID, applicationID, nil
}

func newApplicationsListCommand(opts *applicationsOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List applications in a Docker project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectID, err := resolve.ProjectID(strings.TrimSpace(opts.project))
			if err != nil {
				return err
			}
			client, serverID, err := dockerClient(opts.server)
			if err != nil {
				return err
			}
			applications, err := client.ListDockerApplications(serverID, projectID)
			if err != nil {
				return fmt.Errorf("failed to list Docker applications: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, sanitizedDockerApplications(applications))
			}

			rows := make([][]string, 0, len(applications))
			for _, application := range applications {
				rows = append(rows, []string{
					application.ID,
					application.Name,
					application.SourceType,
					applicationBuild(application),
					fmt.Sprintf("%d", application.InternalPort),
					output.StatusDot(application.Status),
					formatTime(application.LastDeployedAt),
				})
			}
			output.RenderTable("Docker Applications", []string{"ID", "Name", "Source", "Build", "Port", "Status", "Last Deployed"}, rows)
			return nil
		},
	}
}

func newApplicationsShowCommand(opts *applicationsOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "show [application-id]",
		Short: "Show Docker application details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, serverID, projectID, applicationID, err := dockerApplicationContext(opts, args)
			if err != nil {
				return err
			}
			application, err := client.GetDockerApplication(serverID, projectID, applicationID)
			if err != nil {
				return fmt.Errorf("failed to get Docker application: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, sanitizedDockerApplication(*application))
			}

			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), tui.Title.Render(application.Name))
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("ID:")+tui.Value.Render(application.ID))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Project ID:")+tui.Value.Render(application.ProjectID))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Status:")+output.StatusDot(application.Status))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Source:")+tui.Value.Render(applicationSource(application)))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Build:")+tui.Value.Render(applicationBuild(*application)))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Internal Port:")+tui.Value.Render(fmt.Sprintf("%d", application.InternalPort)))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Container:")+tui.Value.Render(application.ContainerName))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Last Deployed:")+tui.Value.Render(formatTime(application.LastDeployedAt)))
			if application.BuildLocation == "github_actions" {
				fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("GHA Ready:")+tui.Value.Render(fmt.Sprintf("%t", application.GHABuildReady)))
				fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("GHA Status:")+tui.Value.Render(application.GHAInstallStatus))
				fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("GHA Out of Sync:")+tui.Value.Render(fmt.Sprintf("%t", application.GHAOutOfSync)))
			}
			fmt.Fprintln(cmd.OutOrStdout())
			return nil
		},
	}
}

func newApplicationsCreateCommand(opts *applicationsOptions) *cobra.Command {
	create := &applicationCreateOptions{port: 80}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Docker application",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateApplicationCreateOptions(*create); err != nil {
				return err
			}
			request, err := buildApplicationCreateRequest(cmd, *create)
			if err != nil {
				return err
			}
			projectID, err := resolve.ProjectID(strings.TrimSpace(opts.project))
			if err != nil {
				return err
			}
			client, serverID, err := dockerClient(opts.server)
			if err != nil {
				return err
			}
			application, err := client.CreateDockerApplication(serverID, projectID, request)
			if err != nil {
				return fmt.Errorf("failed to create Docker application: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, sanitizedDockerApplication(*application))
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("Docker application %q created (ID: %s)", application.Name, application.ID)))
			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&create.name, "name", "", "Application name (required)")
	flags.StringVar(&create.source, "source", "", "Source type: image, git, or dockerfile (required)")
	flags.IntVar(&create.port, "port", 80, "Internal container port")
	flags.StringVar(&create.image, "image", "", "Tagged container image for --source image")
	flags.StringVar(&create.registryCredential, "registry-credential", "", "Saved registry credential ID")
	flags.StringVar(&create.repo, "repo", "", "Git repository URL for --source git")
	flags.StringVar(&create.branch, "branch", "", "Git branch for --source git")
	flags.StringVar(&create.sourceControl, "source-control", "", "Source control connection ID")
	flags.StringVar(&create.buildType, "build-type", "", "Build type: nixpacks or dockerfile")
	flags.StringVar(&create.dockerfilePath, "dockerfile-path", "", "Dockerfile path inside a Git repository")
	flags.StringVar(&create.buildLocation, "build-location", "", "Build location: server or github_actions")
	flags.StringVar(&create.dockerfile, "dockerfile", "", "Inline Dockerfile source path, or - for stdin")
	return cmd
}

func validateApplicationCreateOptions(opts applicationCreateOptions) error {
	if strings.TrimSpace(opts.name) == "" {
		return fmt.Errorf("--name is required")
	}
	if opts.port < 1 || opts.port > 65535 {
		return fmt.Errorf("--port must be between 1 and 65535")
	}

	source := strings.ToLower(strings.TrimSpace(opts.source))
	switch source {
	case "image":
		if strings.TrimSpace(opts.image) == "" {
			return fmt.Errorf("--image is required when --source=image")
		}
		if !hasExplicitImageVersion(opts.image) {
			return fmt.Errorf("--image must include an explicit tag or digest")
		}
		if opts.repo != "" || opts.branch != "" || opts.sourceControl != "" || opts.buildType != "" || opts.dockerfilePath != "" || opts.buildLocation != "" || opts.dockerfile != "" {
			return fmt.Errorf("Git and Dockerfile flags cannot be used when --source=image")
		}
	case "git":
		if strings.TrimSpace(opts.repo) == "" || strings.TrimSpace(opts.branch) == "" {
			return fmt.Errorf("--repo and --branch are required when --source=git")
		}
		if opts.image != "" || opts.registryCredential != "" || opts.dockerfile != "" {
			return fmt.Errorf("image and inline Dockerfile flags cannot be used when --source=git")
		}
		if opts.buildType != "" && opts.buildType != "nixpacks" && opts.buildType != "dockerfile" {
			return fmt.Errorf("--build-type must be nixpacks or dockerfile")
		}
		if opts.dockerfilePath != "" && opts.buildType != "dockerfile" {
			return fmt.Errorf("--dockerfile-path requires --build-type=dockerfile")
		}
		if opts.buildLocation != "" && opts.buildLocation != "server" && opts.buildLocation != "github_actions" {
			return fmt.Errorf("--build-location must be server or github_actions")
		}
		if opts.buildLocation == "github_actions" && strings.TrimSpace(opts.sourceControl) == "" {
			return fmt.Errorf("--build-location=github_actions requires --source-control")
		}
	case "dockerfile":
		if strings.TrimSpace(opts.dockerfile) == "" {
			return fmt.Errorf("--dockerfile is required when --source=dockerfile")
		}
		if opts.image != "" || opts.registryCredential != "" || opts.repo != "" || opts.branch != "" || opts.sourceControl != "" || opts.buildType != "" || opts.dockerfilePath != "" || opts.buildLocation != "" {
			return fmt.Errorf("image and Git flags cannot be used when --source=dockerfile")
		}
	default:
		return fmt.Errorf("--source must be image, git, or dockerfile")
	}
	return nil
}

func hasExplicitImageVersion(value string) bool {
	value = strings.TrimSpace(value)
	if at := strings.LastIndex(value, "@"); at >= 0 {
		digest := value[at+1:]
		separator := strings.Index(digest, ":")
		return separator > 0 && separator < len(digest)-1
	}
	lastSegment := value
	if slash := strings.LastIndex(lastSegment, "/"); slash >= 0 {
		lastSegment = lastSegment[slash+1:]
	}
	colon := strings.LastIndex(lastSegment, ":")
	return colon > 0 && colon < len(lastSegment)-1
}

func buildApplicationCreateRequest(cmd *cobra.Command, opts applicationCreateOptions) (api.CreateDockerApplicationRequest, error) {
	port := opts.port
	request := api.CreateDockerApplicationRequest{
		Name:         strings.TrimSpace(opts.name),
		InternalPort: &port,
		SourceType:   strings.ToLower(strings.TrimSpace(opts.source)),
	}
	switch request.SourceType {
	case "image":
		request.Image = &api.DockerImageSourceInput{
			Image:                strings.TrimSpace(opts.image),
			RegistryCredentialID: optionalString(opts.registryCredential),
		}
	case "git":
		request.Git = &api.DockerGitSourceInput{
			Repo:            strings.TrimSpace(opts.repo),
			Branch:          strings.TrimSpace(opts.branch),
			SourceControlID: optionalString(opts.sourceControl),
			BuildType:       optionalString(opts.buildType),
			DockerfilePath:  optionalString(opts.dockerfilePath),
			BuildLocation:   optionalString(opts.buildLocation),
		}
	case "dockerfile":
		contents, err := readDockerfile(cmd.InOrStdin(), opts.dockerfile)
		if err != nil {
			return api.CreateDockerApplicationRequest{}, err
		}
		request.Dockerfile = &api.DockerfileSourceInput{Contents: contents}
	}
	return request, nil
}

func readDockerfile(stdin io.Reader, path string) (string, error) {
	var reader io.Reader
	var file *os.File
	if strings.TrimSpace(path) == "-" {
		reader = stdin
	} else {
		var err error
		file, err = os.Open(path)
		if err != nil {
			return "", fmt.Errorf("read Dockerfile: %w", err)
		}
		defer file.Close()
		reader = file
	}

	data, err := io.ReadAll(io.LimitReader(reader, maxDockerfileBytes+1))
	if err != nil {
		return "", fmt.Errorf("read Dockerfile: %w", err)
	}
	if len(data) > maxDockerfileBytes {
		return "", fmt.Errorf("Dockerfile exceeds %d KiB limit", maxDockerfileBytes>>10)
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("Dockerfile cannot be empty")
	}
	return string(data), nil
}

func newApplicationsUpdateCommand(opts *applicationsOptions) *cobra.Command {
	var name, buildType, dockerfilePath string
	cmd := &cobra.Command{
		Use:   "update [application-id]",
		Short: "Update a Docker application",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("build-type") && !cmd.Flags().Changed("dockerfile-path") {
				return fmt.Errorf("at least one of --name, --build-type, or --dockerfile-path is required")
			}
			req := api.UpdateDockerApplicationRequest{}
			if cmd.Flags().Changed("name") {
				trimmed := strings.TrimSpace(name)
				if trimmed == "" {
					return fmt.Errorf("--name cannot be empty")
				}
				req.Name = &trimmed
			}
			if cmd.Flags().Changed("build-type") {
				trimmed := strings.TrimSpace(buildType)
				if trimmed != "auto" && trimmed != "nixpacks" && trimmed != "dockerfile" {
					return fmt.Errorf("--build-type must be auto, nixpacks, or dockerfile")
				}
				req.BuildType = &trimmed
			}
			if cmd.Flags().Changed("dockerfile-path") {
				trimmed := strings.TrimSpace(dockerfilePath)
				req.DockerfilePath = &trimmed
			}
			if err := validateDockerApplicationUpdateRequest(req); err != nil {
				return err
			}

			client, serverID, projectID, applicationID, err := dockerApplicationContext(opts, args)
			if err != nil {
				return err
			}
			application, err := client.UpdateDockerApplication(serverID, projectID, applicationID, req)
			if err != nil {
				return fmt.Errorf("failed to update Docker application: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, sanitizedDockerApplication(*application))
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("Docker application %q updated", application.Name)))
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New application name")
	cmd.Flags().StringVar(&buildType, "build-type", "", "Build type: auto, nixpacks, or dockerfile")
	cmd.Flags().StringVar(&dockerfilePath, "dockerfile-path", "", "Dockerfile path within the Git repository")
	return cmd
}

func validateDockerApplicationUpdateRequest(req api.UpdateDockerApplicationRequest) error {
	if req.DockerfilePath != nil && *req.DockerfilePath != "" && (req.BuildType == nil || *req.BuildType != "dockerfile") {
		return fmt.Errorf("--dockerfile-path requires --build-type=dockerfile in the same command")
	}
	return nil
}

func newApplicationsDeployCommand(opts *applicationsOptions) *cobra.Command {
	var wait bool
	var timeout int
	cmd := &cobra.Command{
		Use:   "deploy [application-id]",
		Short: "Deploy or rebuild a Docker application",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateDockerWaitOptions(wait, timeout); err != nil {
				return err
			}
			client, serverID, projectID, applicationID, err := dockerApplicationContext(opts, args)
			if err != nil {
				return err
			}
			application, err := client.GetDockerApplication(serverID, projectID, applicationID)
			if err != nil {
				return fmt.Errorf("failed to get Docker application: %w", err)
			}
			deployment, err := client.DeployDockerApplication(serverID, projectID, application.ID)
			if err != nil {
				return fmt.Errorf("failed to deploy Docker application: %w", err)
			}

			if !wait {
				if jsonEnabled(cmd) {
					return writeJSON(cmd, deployment)
				}
				fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("Deployment %s queued for %q", deployment.ID, application.Name)))
				return nil
			}

			if !jsonEnabled(cmd) {
				fmt.Fprintln(cmd.OutOrStdout(), tui.Info.Render(fmt.Sprintf("Deployment %s queued; waiting for completion...", deployment.ID)))
			}
			waitContext, cancel := context.WithTimeout(cmd.Context(), time.Duration(timeout)*time.Second)
			defer cancel()
			final, err := waitForDockerDeployment(waitContext, client, serverID, projectID, application.ID, deployment.ID)
			if err != nil {
				return err
			}
			if _, err := waitForDockerApplicationRunning(waitContext, client, serverID, projectID, application.ID); err != nil {
				return fmt.Errorf("deployment %s succeeded but application reconciliation failed: %w", final.ID, err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, final)
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("Deployment %s completed successfully", final.ID)))
			return nil
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the deployment to finish")
	cmd.Flags().IntVar(&timeout, "timeout", 600, "Wait timeout in seconds")
	return cmd
}

func validateDockerWaitOptions(wait bool, timeout int) error {
	if wait && timeout <= 0 {
		return fmt.Errorf("--timeout must be greater than zero when --wait is used")
	}
	return nil
}

func waitForDockerDeployment(ctx context.Context, client *api.Client, serverID, projectID, applicationID, deploymentID string) (*api.DockerDeploymentResponse, error) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	lastStatus := "pending"
	for {
		deployments, err := client.ListDockerApplicationDeploymentsContext(ctx, serverID, projectID, applicationID)
		if err != nil {
			if ctx.Err() != nil {
				return nil, dockerWaitContextError(ctx, "deployment", lastStatus)
			}
			return nil, fmt.Errorf("failed to check Docker deployment status: %w", err)
		}
		for index := range deployments {
			deployment := &deployments[index]
			if deployment.ID != deploymentID {
				continue
			}
			lastStatus = deployment.Status
			terminal, err := dockerDeploymentOutcome(deployment)
			if terminal {
				return deployment, err
			}
			break
		}

		select {
		case <-ctx.Done():
			return nil, dockerWaitContextError(ctx, "deployment", lastStatus)
		case <-ticker.C:
		}
	}
}

func waitForDockerApplicationRunning(ctx context.Context, client *api.Client, serverID, projectID, applicationID string) (*api.DockerApplicationResponse, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastStatus := "unknown"
	for {
		application, err := client.GetDockerApplicationContext(ctx, serverID, projectID, applicationID)
		if err != nil {
			if ctx.Err() != nil {
				return nil, dockerWaitContextError(ctx, "application reconciliation", lastStatus)
			}
			return nil, err
		}
		lastStatus = application.Status
		switch application.Status {
		case "running":
			return application, nil
		case "failed", "deleting":
			return nil, fmt.Errorf("application entered %q status", application.Status)
		}

		select {
		case <-ctx.Done():
			return nil, dockerWaitContextError(ctx, "application reconciliation", lastStatus)
		case <-ticker.C:
		}
	}
}

func dockerWaitContextError(ctx context.Context, phase, lastStatus string) error {
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("Docker %s timed out (status: %s): %w", phase, lastStatus, ctx.Err())
	}
	return fmt.Errorf("Docker %s cancelled (status: %s): %w", phase, lastStatus, ctx.Err())
}

func dockerDeploymentOutcome(deployment *api.DockerDeploymentResponse) (bool, error) {
	switch deployment.Status {
	case "success":
		return true, nil
	case "failed":
		message := valueOrEmpty(deployment.Error)
		if message == "" {
			message = "deployment failed"
		}
		return true, fmt.Errorf("Docker deployment %s failed: %s", deployment.ID, message)
	case "cancelled":
		return true, fmt.Errorf("Docker deployment %s was cancelled", deployment.ID)
	default:
		return false, nil
	}
}

func newApplicationActionCommand(opts *applicationsOptions, action string) *cobra.Command {
	return &cobra.Command{
		Use:   action + " [application-id]",
		Short: applicationActionDescription(action),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, serverID, projectID, applicationID, err := dockerApplicationContext(opts, args)
			if err != nil {
				return err
			}
			application, err := client.GetDockerApplication(serverID, projectID, applicationID)
			if err != nil {
				return fmt.Errorf("failed to get Docker application: %w", err)
			}
			if err := client.DockerApplicationAction(serverID, projectID, application.ID, action); err != nil {
				return fmt.Errorf("failed to %s Docker application: %w", action, err)
			}
			result := map[string]string{
				"action":         action,
				"application_id": application.ID,
				"status":         "queued",
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, result)
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("%s queued for Docker application %q", applicationActionLabel(action), application.Name)))
			return nil
		},
	}
}

func applicationActionDescription(action string) string {
	switch action {
	case "reload":
		return "Recreate a Docker application with its saved runtime configuration"
	case "start":
		return "Start a stopped Docker application"
	case "stop":
		return "Stop a Docker application"
	default:
		return action + " a Docker application"
	}
}

func applicationActionLabel(action string) string {
	switch action {
	case "reload":
		return "Reload"
	case "start":
		return "Start"
	case "stop":
		return "Stop"
	default:
		return action
	}
}

func newApplicationsDeploymentsCommand(opts *applicationsOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "deployments [application-id]",
		Short: "List Docker application deployments",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, serverID, projectID, applicationID, err := dockerApplicationContext(opts, args)
			if err != nil {
				return err
			}
			deployments, err := client.ListDockerApplicationDeployments(serverID, projectID, applicationID)
			if err != nil {
				return fmt.Errorf("failed to list Docker application deployments: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, deployments)
			}
			rows := make([][]string, 0, len(deployments))
			for _, deployment := range deployments {
				reference := valueOrEmpty(deployment.CommitSHA)
				if reference == "" {
					reference = valueOrEmpty(deployment.ImageRef)
				}
				rows = append(rows, []string{
					deployment.ID,
					output.StatusDot(deployment.Status),
					deployment.TriggerSource,
					reference,
					valueOrEmpty(deployment.TaskID),
					formatTime(deployment.CreatedAt),
					formatTime(deployment.FinishedAt),
				})
			}
			output.RenderTable("Docker Application Deployments", []string{"ID", "Status", "Trigger", "Commit/Image", "Task", "Created", "Finished"}, rows)
			return nil
		},
	}
}

func newApplicationsDeleteCommand(opts *applicationsOptions) *cobra.Command {
	var yes, removeVolumes bool
	cmd := &cobra.Command{
		Use:   "delete [application-id]",
		Short: "Delete a Docker application",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, serverID, projectID, applicationID, err := dockerApplicationContext(opts, args)
			if err != nil {
				return err
			}
			application, err := client.GetDockerApplication(serverID, projectID, applicationID)
			if err != nil {
				return fmt.Errorf("failed to get Docker application: %w", err)
			}

			volumeNotice := "Named volumes will be preserved."
			if removeVolumes {
				volumeNotice = "Named volumes will also be permanently removed."
			}
			confirmed, err := confirmDestructive(
				cmd,
				yes,
				fmt.Sprintf("Delete Docker application %q?", application.Name),
				fmt.Sprintf("Application ID: %s. %s", application.ID, volumeNotice),
			)
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Fprintln(cmd.OutOrStdout(), tui.Dim.Render("Cancelled"))
				return nil
			}
			if err := client.DeleteDockerApplication(serverID, projectID, application.ID, removeVolumes); err != nil {
				return fmt.Errorf("failed to delete Docker application: %w", err)
			}
			result := map[string]any{
				"application_id": application.ID,
				"remove_volumes": removeVolumes,
				"status":         "deleting",
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, result)
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("Deletion queued for Docker application %q. %s", application.Name, volumeNotice)))
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm deletion without prompting")
	cmd.Flags().BoolVar(&removeVolumes, "remove-volumes", false, "Permanently remove the application's named volumes")
	return cmd
}

func applicationBuild(application api.DockerApplicationResponse) string {
	switch application.SourceType {
	case "image":
		return "pre-built"
	case "dockerfile":
		return "dockerfile"
	}
	buildType := valueOrEmpty(application.BuildType)
	if buildType == "" {
		buildType = "auto"
	}
	if application.BuildLocation == "" || application.SourceType != "git" {
		return buildType
	}
	return buildType + " / " + application.BuildLocation
}

func applicationSource(application *api.DockerApplicationResponse) string {
	switch application.SourceType {
	case "image":
		if value, ok := application.SourceConfig["image"].(string); ok && value != "" {
			return redact.URL(value)
		}
	case "git":
		repo, _ := application.SourceConfig["repo"].(string)
		repo = redact.URL(repo)
		branch, _ := application.SourceConfig["branch"].(string)
		if repo != "" && branch != "" {
			return repo + " @ " + branch
		}
		if repo != "" {
			return repo
		}
	case "dockerfile":
		return "inline Dockerfile"
	}
	return application.SourceType
}

func sanitizedDockerApplications(applications []api.DockerApplicationResponse) []api.DockerApplicationResponse {
	result := make([]api.DockerApplicationResponse, len(applications))
	for index := range applications {
		result[index] = sanitizedDockerApplication(applications[index])
	}
	return result
}

func sanitizedDockerApplication(application api.DockerApplicationResponse) api.DockerApplicationResponse {
	if application.SourceConfig == nil {
		return application
	}
	sourceConfig := application.SourceConfig
	application.SourceConfig = make(map[string]any, len(sourceConfig))
	for key, value := range sourceConfig {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "token") || strings.Contains(lowerKey, "password") {
			application.SourceConfig[key] = "REDACTED"
			continue
		}
		if lowerKey == "repo" || lowerKey == "image" || lowerKey == "registry_url" {
			if rawURL, ok := value.(string); ok {
				value = redact.URL(rawURL)
			}
		}
		application.SourceConfig[key] = value
	}
	return application
}
