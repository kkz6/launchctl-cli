package nav

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/notify"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/redact"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/kkz6/launchctl/internal/tui/logview"
)

func dockerProjectsMenu(client *api.Client, cfg *config.Config, serverID, serverName string) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Projects")

		projects, err := client.ListDockerProjects(serverID)
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to list Docker projects: %s", err))
			tui.WaitForEnter()
			return
		}

		if len(projects) == 0 {
			tui.ShowInfo("No Docker projects found on this server")
			fmt.Println(tui.Dim.Render("Create one with: lctl docker projects create --server " + serverID + " --name <name>"))
			tui.WaitForEnter()
			return
		}

		columns := []tui.TableColumn{
			{Header: "Name", Width: 24},
			{Header: "Apps", Width: 7},
			{Header: "Composes", Width: 10},
			{Header: "Databases", Width: 11},
			{Header: "Description", Width: 32},
		}
		rows := make([]tui.TableRow, 0, len(projects))
		for _, project := range projects {
			description := ""
			if project.Description != nil {
				description = truncate(*project.Description, 30)
			}
			rows = append(rows, tui.TableRow{Columns: []string{
				project.Name,
				fmt.Sprintf("%d", project.ApplicationsCount),
				fmt.Sprintf("%d", project.ComposesCount),
				fmt.Sprintf("%d", project.DatabasesCount),
				description,
			}})
		}

		choice, err := tui.SelectFromTable("Select a project", columns, rows, "Back")
		if err != nil || choice == len(rows) {
			return
		}
		dockerProjectActions(client, cfg, serverID, serverName, projects[choice])
	}
}

func dockerProjectActions(client *api.Client, cfg *config.Config, serverID, serverName string, project api.DockerProjectResponse) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Projects", project.Name)

		choice, err := tui.SelectFromList(
			fmt.Sprintf("Project: %s", project.Name),
			[]string{"Show Details", "View Applications"},
			"Back",
		)
		if err != nil || choice == 2 {
			return
		}

		switch choice {
		case 0:
			showDockerProjectDetails(serverName, project)
		case 1:
			dockerApplicationsMenu(client, cfg, serverID, serverName, project)
		}
	}
}

func showDockerProjectDetails(serverName string, project api.DockerProjectResponse) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Projects", project.Name, "Details")

	fmt.Println(tui.Label.Render("ID:") + tui.Value.Render(project.ID))
	fmt.Println(tui.Label.Render("Applications:") + tui.Value.Render(fmt.Sprintf("%d", project.ApplicationsCount)))
	fmt.Println(tui.Label.Render("Composes:") + tui.Value.Render(fmt.Sprintf("%d", project.ComposesCount)))
	fmt.Println(tui.Label.Render("Databases:") + tui.Value.Render(fmt.Sprintf("%d", project.DatabasesCount)))
	if project.Description != nil && strings.TrimSpace(*project.Description) != "" {
		fmt.Println(tui.Label.Render("Description:") + tui.Value.Render(*project.Description))
	}
	if project.CreatedAt != nil {
		fmt.Println(tui.Label.Render("Created:") + tui.Value.Render(formatDockerTime(project.CreatedAt)))
	}

	tui.WaitForEnter()
}

func dockerApplicationsMenu(client *api.Client, cfg *config.Config, serverID, serverName string, project api.DockerProjectResponse) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Projects", project.Name, "Applications")

		applications, err := client.ListDockerApplications(serverID, project.ID)
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to list Docker applications: %s", err))
			tui.WaitForEnter()
			return
		}

		if len(applications) == 0 {
			tui.ShowInfo("No applications found in this project")
			fmt.Println(tui.Dim.Render("Create one with: lctl docker applications create --server " + serverID + " --project " + project.ID))
			tui.WaitForEnter()
			return
		}

		columns := []tui.TableColumn{
			{Header: "Name", Width: 20},
			{Header: "Source", Width: 10},
			{Header: "Build", Width: 14},
			{Header: "Port", Width: 6},
			{Header: "Status", Width: 14},
			{Header: "Last deployed", Width: 20},
		}
		rows := make([]tui.TableRow, 0, len(applications))
		for _, application := range applications {
			rows = append(rows, tui.TableRow{Columns: []string{
				application.Name,
				application.SourceType,
				dockerBuildSummary(application),
				fmt.Sprintf("%d", application.InternalPort),
				output.StatusDot(application.Status),
				formatDockerTime(application.LastDeployedAt),
			}})
		}

		choice, err := tui.SelectFromTable("Select an application", columns, rows, "Back")
		if err != nil || choice == len(rows) {
			return
		}
		dockerApplicationActions(client, cfg, serverID, serverName, project, applications[choice])
	}
}

type dockerApplicationMenuAction string

const (
	dockerApplicationDetails     dockerApplicationMenuAction = "details"
	dockerApplicationDeploy      dockerApplicationMenuAction = "deploy"
	dockerApplicationDeployments dockerApplicationMenuAction = "deployments"
	dockerApplicationReload      dockerApplicationMenuAction = "reload"
	dockerApplicationStart       dockerApplicationMenuAction = "start"
	dockerApplicationStop        dockerApplicationMenuAction = "stop"
)

type dockerApplicationActionOption struct {
	key   dockerApplicationMenuAction
	label string
}

func dockerApplicationActionOptions(application api.DockerApplicationResponse) []dockerApplicationActionOption {
	options := []dockerApplicationActionOption{{key: dockerApplicationDetails, label: "Show Details"}}
	busy := false
	switch application.Status {
	case "building", "deleting", "starting", "stopping", "restarting":
		busy = true
	}
	if !busy {
		options = append(options, dockerApplicationActionOption{key: dockerApplicationDeploy, label: "Deploy"})
	}
	options = append(options, dockerApplicationActionOption{key: dockerApplicationDeployments, label: "View Deployments"})
	if application.LastDeployedAt == nil || busy {
		return options
	}

	options = append(options, dockerApplicationActionOption{key: dockerApplicationReload, label: "Reload"})
	switch application.Status {
	case "running":
		options = append(options, dockerApplicationActionOption{key: dockerApplicationStop, label: "Stop"})
	case "stopped":
		options = append(options, dockerApplicationActionOption{key: dockerApplicationStart, label: "Start"})
	}
	return options
}

func dockerApplicationActions(client *api.Client, cfg *config.Config, serverID, serverName string, project api.DockerProjectResponse, application api.DockerApplicationResponse) {
	for {
		fresh, err := client.GetDockerApplication(serverID, project.ID, application.ID)
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to refresh Docker application: %s", err))
			tui.WaitForEnter()
			return
		}
		application = *fresh

		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Projects", project.Name, "Applications", application.Name)

		actions := dockerApplicationActionOptions(application)
		labels := make([]string, 0, len(actions))
		for _, action := range actions {
			labels = append(labels, action.label)
		}
		choice, err := tui.SelectFromList(
			fmt.Sprintf("Application: %s", application.Name),
			labels,
			"Back",
		)
		if err != nil || choice == len(actions) {
			return
		}

		switch actions[choice].key {
		case dockerApplicationDetails:
			showDockerApplicationDetails(serverName, project.Name, application)
		case dockerApplicationDeploy:
			deployDockerApplication(client, cfg, serverID, serverName, project, application)
		case dockerApplicationDeployments:
			viewDockerApplicationDeployments(client, cfg, serverID, serverName, project, application)
		case dockerApplicationReload:
			runDockerApplicationAction(client, serverID, project.ID, application, "reload")
		case dockerApplicationStart:
			runDockerApplicationAction(client, serverID, project.ID, application, "start")
		case dockerApplicationStop:
			runDockerApplicationAction(client, serverID, project.ID, application, "stop")
		}
	}
}

func showDockerApplicationDetails(serverName, projectName string, application api.DockerApplicationResponse) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Projects", projectName, "Applications", application.Name, "Details")

	fmt.Println(tui.Label.Render("ID:") + tui.Value.Render(application.ID))
	fmt.Println(tui.Label.Render("Status:") + output.StatusDot(application.Status))
	fmt.Println(tui.Label.Render("Source:") + tui.Value.Render(application.SourceType))
	if summary := dockerSourceSummary(application); summary != "" {
		fmt.Println(tui.Label.Render("Source detail:") + tui.Value.Render(summary))
	}
	fmt.Println(tui.Label.Render("Internal port:") + tui.Value.Render(fmt.Sprintf("%d", application.InternalPort)))
	fmt.Println(tui.Label.Render("Build:") + tui.Value.Render(dockerBuildSummary(application)))
	if application.ContainerName != "" {
		fmt.Println(tui.Label.Render("Container:") + tui.Value.Render(application.ContainerName))
	}
	if application.LastDeployedAt != nil {
		fmt.Println(tui.Label.Render("Last deployed:") + tui.Value.Render(formatDockerTime(application.LastDeployedAt)))
	}
	if application.BuildLocation == "github_actions" {
		fmt.Println(tui.Label.Render("GHA ready:") + tui.Value.Render(fmt.Sprintf("%t", application.GHABuildReady)))
		if application.GHAInstallStatus != "" {
			fmt.Println(tui.Label.Render("GHA status:") + tui.Value.Render(application.GHAInstallStatus))
		}
		if application.GHAOutOfSync {
			fmt.Println(tui.Label.Render("Pending sync:") + tui.Value.Render(fmt.Sprintf("%d changes", application.GHAPendingChanges)))
		}
	}

	tui.WaitForEnter()
}

func deployDockerApplication(client *api.Client, cfg *config.Config, serverID, serverName string, project api.DockerProjectResponse, application api.DockerApplicationResponse) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Projects", project.Name, "Applications", application.Name, "Deploy")

	var confirm bool
	if err := huh.NewConfirm().
		Title(fmt.Sprintf("Deploy %q?", application.Name)).
		Description("This rebuilds the application and replaces its running container.").
		Value(&confirm).
		WithTheme(tui.FormTheme()).
		Run(); err != nil || !confirm {
		fmt.Println(tui.Dim.Render("Cancelled"))
		tui.WaitForEnter()
		return
	}

	deployment, err := client.DeployDockerApplication(serverID, project.ID, application.ID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to start deployment: %s", err))
		tui.WaitForEnter()
		return
	}

	tui.ShowSuccess(fmt.Sprintf("Deployment %s queued (%s)", deployment.ID, deployment.Status))
	notify.Success(fmt.Sprintf("Deployment of %s queued", application.Name))
	if application.BuildLocation != "github_actions" && deployment.TaskID == nil {
		tui.ShowInfo("Waiting for deployment task logs...")
		latest, err := waitForDockerDeploymentTask(client, serverID, project.ID, application.ID, *deployment, 15*time.Second)
		if err != nil {
			tui.ShowWarning(fmt.Sprintf("Could not attach live logs: %s", err))
		} else {
			deployment = &latest
		}
	}
	if deployment.TaskID == nil {
		if isTerminalDockerDeployment(deployment.Status) {
			showDockerDeploymentDetails(serverName, project.Name, application.Name, *deployment)
			return
		}
		tui.ShowInfo("The deployment is running in the background")
		tui.WaitForEnter()
		return
	}
	viewDockerDeployment(client, cfg, serverID, serverName, project, application, *deployment)
}

func waitForDockerDeploymentTask(client *api.Client, serverID, projectID, applicationID string, deployment api.DockerDeploymentResponse, timeout time.Duration) (api.DockerDeploymentResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		deployments, err := client.ListDockerApplicationDeploymentsContext(ctx, serverID, projectID, applicationID)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return deployment, nil
			}
			return deployment, err
		}
		for _, candidate := range deployments {
			if candidate.ID != deployment.ID {
				continue
			}
			deployment = candidate
			if deployment.TaskID != nil || isTerminalDockerDeployment(deployment.Status) {
				return deployment, nil
			}
			break
		}

		select {
		case <-ctx.Done():
			return deployment, nil
		case <-ticker.C:
		}
	}
}

func isTerminalDockerDeployment(status string) bool {
	switch status {
	case "success", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func runDockerApplicationAction(client *api.Client, serverID, projectID string, application api.DockerApplicationResponse, action string) {
	if action == "reload" || action == "stop" {
		description := "This stops the running application container."
		if action == "reload" {
			description = "This recreates the container from its current image and applies saved runtime configuration."
		}
		confirmed := false
		if err := huh.NewConfirm().
			Title(fmt.Sprintf("%s %q?", strings.ToUpper(action[:1])+action[1:], application.Name)).
			Description(description).
			Value(&confirmed).
			WithTheme(tui.FormTheme()).
			Run(); err != nil || !confirmed {
			fmt.Println(tui.Dim.Render("Cancelled"))
			tui.WaitForEnter()
			return
		}
	}

	if err := client.DockerApplicationAction(serverID, projectID, application.ID, action); err != nil {
		tui.ShowError(fmt.Sprintf("Failed to %s application: %s", action, err))
		tui.WaitForEnter()
		return
	}
	verb := action
	if action == "reload" {
		verb = "reload (recreate with current configuration)"
	}
	tui.ShowSuccess(fmt.Sprintf("Application %s queued", verb))
	notify.Success(fmt.Sprintf("%s queued for %s", action, application.Name))
	tui.WaitForEnter()
}

func viewDockerApplicationDeployments(client *api.Client, cfg *config.Config, serverID, serverName string, project api.DockerProjectResponse, application api.DockerApplicationResponse) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Projects", project.Name, "Applications", application.Name, "Deployments")

		deployments, err := client.ListDockerApplicationDeployments(serverID, project.ID, application.ID)
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to list deployments: %s", err))
			tui.WaitForEnter()
			return
		}
		if len(deployments) == 0 {
			tui.ShowInfo("No deployments found")
			tui.WaitForEnter()
			return
		}

		columns := []tui.TableColumn{
			{Header: "Status", Width: 16},
			{Header: "Trigger", Width: 16},
			{Header: "Commit / image", Width: 30},
			{Header: "Created", Width: 22},
		}
		rows := make([]tui.TableRow, 0, len(deployments))
		for _, deployment := range deployments {
			rows = append(rows, tui.TableRow{Columns: []string{
				output.StatusDot(deployment.Status),
				deployment.TriggerSource,
				dockerDeploymentSource(deployment),
				formatDockerTime(deployment.CreatedAt),
			}})
		}

		choice, err := tui.SelectFromTable("Select a deployment", columns, rows, "Back")
		if err != nil || choice == len(rows) {
			return
		}
		viewDockerDeployment(client, cfg, serverID, serverName, project, application, deployments[choice])
	}
}

func viewDockerDeployment(client *api.Client, cfg *config.Config, serverID, serverName string, project api.DockerProjectResponse, application api.DockerApplicationResponse, deployment api.DockerDeploymentResponse) {
	if deployment.TaskID == nil {
		showDockerDeploymentDetails(serverName, project.Name, application.Name, deployment)
		return
	}

	jwt, err := client.ExchangeToken()
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to authenticate for deployment logs: %s", err))
		tui.WaitForEnter()
		return
	}
	ws, err := api.NewLogsWSClient(cfg, jwt, serverID, "task", *deployment.TaskID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to connect to deployment logs: %s", err))
		tui.WaitForEnter()
		return
	}
	defer ws.Close()

	info := logview.Info{
		Title:  fmt.Sprintf("Deployment: %s", application.Name),
		Status: deployment.Status,
	}
	if deployment.CommitSHA != nil {
		info.Commit = *deployment.CommitSHA
		info.Lines = append(info.Lines, struct{ Label, Value string }{"Commit", *deployment.CommitSHA})
	}
	if deployment.ImageRef != nil {
		info.Lines = append(info.Lines, struct{ Label, Value string }{"Image", *deployment.ImageRef})
	}
	info.Lines = append(info.Lines,
		struct{ Label, Value string }{"Trigger", deployment.TriggerSource},
		struct{ Label, Value string }{"Created", formatDockerTime(deployment.CreatedAt)},
	)

	if err := logview.Run(info, ws); err != nil {
		tui.ShowError(fmt.Sprintf("Deployment log viewer error: %s", err))
		tui.WaitForEnter()
	}
}

func showDockerDeploymentDetails(serverName, projectName, applicationName string, deployment api.DockerDeploymentResponse) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Projects", projectName, "Applications", applicationName, "Deployment")
	fmt.Println(tui.Label.Render("ID:") + tui.Value.Render(deployment.ID))
	fmt.Println(tui.Label.Render("Status:") + output.StatusDot(deployment.Status))
	fmt.Println(tui.Label.Render("Trigger:") + tui.Value.Render(deployment.TriggerSource))
	if source := dockerDeploymentSource(deployment); source != "" {
		fmt.Println(tui.Label.Render("Source:") + tui.Value.Render(source))
	}
	if deployment.GHARunURL != nil {
		fmt.Println(tui.Label.Render("GitHub run:") + tui.Value.Render(*deployment.GHARunURL))
	}
	if deployment.Error != nil {
		fmt.Println(tui.Label.Render("Error:") + tui.Error.Render(*deployment.Error))
	}
	fmt.Println(tui.Label.Render("Created:") + tui.Value.Render(formatDockerTime(deployment.CreatedAt)))
	if deployment.FinishedAt != nil {
		fmt.Println(tui.Label.Render("Finished:") + tui.Value.Render(formatDockerTime(deployment.FinishedAt)))
	}
	tui.WaitForEnter()
}

func dockerBuildSummary(application api.DockerApplicationResponse) string {
	buildType := "pre-built"
	if application.SourceType == "git" {
		buildType = "auto"
		if application.BuildType != nil && *application.BuildType != "" {
			buildType = *application.BuildType
		}
	} else if application.SourceType == "dockerfile" {
		buildType = "dockerfile"
	}
	if application.BuildLocation == "github_actions" {
		return buildType + " / GHA"
	}
	return buildType
}

func dockerSourceSummary(application api.DockerApplicationResponse) string {
	keys := []string{"image", "repo", "branch"}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		if value, ok := application.SourceConfig[key].(string); ok && value != "" {
			if key == "repo" || key == "image" {
				value = redact.URL(value)
			}
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " @ ")
}

func dockerDeploymentSource(deployment api.DockerDeploymentResponse) string {
	if deployment.CommitSHA != nil && *deployment.CommitSHA != "" {
		return *deployment.CommitSHA
	}
	if deployment.ImageRef != nil {
		return *deployment.ImageRef
	}
	return ""
}

func formatDockerTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Local().Format("2006-01-02 15:04:05")
}
