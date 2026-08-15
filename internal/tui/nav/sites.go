package nav

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/notify"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/tui"
	deploytui "github.com/kkz6/launchctl/internal/tui/deploy"
	"github.com/kkz6/launchctl/internal/tui/logview"
)

func sitesMenu(client *api.Client, cfg *config.Config, serverID, serverName string) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Sites")

		sites, err := client.ListSites(serverID)
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to list sites: %s", err))
			tui.WaitForEnter()
			return
		}

		if len(sites) == 0 {
			tui.ShowInfo("No sites found on this server")
			tui.WaitForEnter()
			return
		}

		columns := []tui.TableColumn{
			{Header: "Address", Width: 26},
			{Header: "Type", Width: 10},
			{Header: "Branch", Width: 14},
			{Header: "Status", Width: 14},
			{Header: "Deploy", Width: 12},
			{Header: "PHP", Width: 8},
		}

		var rows []tui.TableRow
		for _, s := range sites {
			deployStatus := ""
			if s.LatestDeployment != nil {
				deployStatus = s.LatestDeployment.Status
			}
			rows = append(rows, tui.TableRow{
				Columns: []string{s.Address, s.TypeLabel(), s.RepositoryBranch, output.StatusDot(s.Status), deployStatus, s.PHPVersion},
			})
		}

		choice, err := tui.SelectFromTable("Select a site", columns, rows, "Back")
		if err != nil || choice == len(rows) {
			return
		}

		siteActions(client, cfg, serverID, serverName, sites[choice])
	}
}

func siteActions(client *api.Client, cfg *config.Config, serverID, serverName string, site api.SiteResponse) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Sites", site.Address)

		favLabel := "★ Add to Favorites"
		if cfg.IsFavorite(site.ID) {
			favLabel = "☆ Remove from Favorites"
		}

		choice, err := tui.SelectFromList(
			fmt.Sprintf("Site: %s", site.Address),
			[]string{"Show Details", "Deploy", "View Deployments", "Environment", "View Logs", "Run Command", favLabel},
			"Back",
		)
		if err != nil || choice == 7 {
			return
		}

		switch choice {
		case 0:
			showSiteDetails(serverName, site)
		case 1:
			deploySite(client, cfg, serverID, serverName, site)
		case 2:
			viewDeployments(client, cfg, serverID, serverName, site)
		case 3:
			viewEnvFile(client, serverID, site.ID, serverName, site.Address)
		case 4:
			viewSiteLogs(client, serverID, site.ID, serverName, site.Address)
		case 5:
			runSiteCommand(client, serverID, site.ID, serverName, site.Address)
		case 6:
			if cfg.IsFavorite(site.ID) {
				cfg.RemoveFavorite(site.ID)
				notify.Success(fmt.Sprintf("Removed %s from favorites", site.Address))
			} else {
				cfg.AddFavorite(config.Favorite{
					ServerID:    serverID,
					ServerName:  serverName,
					SiteID:      site.ID,
					SiteAddress: site.Address,
				})
				notify.Success(fmt.Sprintf("Added %s to favorites", site.Address))
			}
		}
	}
}

func showSiteDetails(serverName string, site api.SiteResponse) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Sites", site.Address, "Details")

	fmt.Println(tui.Label.Render("ID:") + tui.Value.Render(site.ID))
	fmt.Println(tui.Label.Render("Status:") + output.StatusDot(site.Status))
	fmt.Println(tui.Label.Render("Type:") + tui.Value.Render(site.TypeLabel()))
	fmt.Println(tui.Label.Render("URL:") + tui.Value.Render(site.URL))
	fmt.Println(tui.Label.Render("Path:") + tui.Value.Render(site.Path))
	fmt.Println(tui.Label.Render("PHP:") + tui.Value.Render(site.PHPVersion))
	fmt.Println(tui.Label.Render("Branch:") + tui.Value.Render(site.RepositoryBranch))
	fmt.Println(tui.Label.Render("Zero Downtime:") + tui.Value.Render(fmt.Sprintf("%v", site.ZeroDowntimeDeployment)))
	fmt.Println(tui.Label.Render("Auto Deploy:") + tui.Value.Render(fmt.Sprintf("%v", site.AutoDeployment)))

	if site.RepositoryURL != nil {
		fmt.Println(tui.Label.Render("Repository:") + tui.Value.Render(*site.RepositoryURL))
	}

	if site.LatestDeployment != nil {
		fmt.Println()
		fmt.Println(tui.Subtitle.Render("Latest Deployment"))
		fmt.Println(tui.Label.Render("Status:") + output.StatusDot(site.LatestDeployment.Status))
		fmt.Println(tui.Label.Render("Commit:") + tui.Value.Render(site.LatestDeployment.ShortGitHash))
		if site.LatestDeployment.CommitData != nil {
			fmt.Println(tui.Label.Render("Message:") + tui.Value.Render(site.LatestDeployment.CommitData.Message))
		}
		fmt.Println(tui.Label.Render("Created:") + tui.Dim.Render(site.LatestDeployment.CreatedAt))
	}

	tui.WaitForEnter()
}

func deploySite(client *api.Client, cfg *config.Config, serverID, serverName string, site api.SiteResponse) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Sites", site.Address, "Deploy")

	var confirm bool
	huh.NewConfirm().
		Title(fmt.Sprintf("Deploy %q?", site.Address)).
		Description("This will trigger a new deployment.").
		Value(&confirm).
		Run()

	if !confirm {
		fmt.Println(tui.Dim.Render("Cancelled"))
		tui.WaitForEnter()
		return
	}

	deployment, err := client.DeploySite(serverID, site.ID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to trigger deployment: %s", err))
		tui.WaitForEnter()
		return
	}

	tui.ShowSuccess(fmt.Sprintf("Deployment %s triggered", deployment.ID))

	jwt, err := client.ExchangeToken()
	if err != nil {
		tui.ShowWarning("Could not authenticate for live logs, deployment is running in the background")
		tui.WaitForEnter()
		return
	}

	ws, err := api.NewWSClient(cfg, jwt)
	if err != nil {
		tui.ShowWarning("Could not connect to live logs, deployment is running in the background")
		tui.WaitForEnter()
		return
	}
	defer ws.Close()

	teamChannel := fmt.Sprintf("team.%s", cfg.TeamID)
	if err := ws.Subscribe(teamChannel); err != nil {
		tui.ShowWarning("Could not subscribe to events")
		tui.WaitForEnter()
		return
	}

	model := deploytui.NewModel(deploytui.Opts{
		SiteName:     site.Address,
		ServerID:     serverID,
		SiteID:       site.ID,
		DeploymentID: deployment.ID,
		Client:       client,
		JWT:          jwt,
		TeamID:       cfg.TeamID,
		APIURL:       client.BaseURL(),
		WS:           ws,
	})
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		tui.ShowError(fmt.Sprintf("Deploy view error: %s", err))
		tui.WaitForEnter()
	}
}

func viewDeployments(client *api.Client, cfg *config.Config, serverID, serverName string, site api.SiteResponse) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Sites", site.Address, "Deployments")

		deployments, err := client.ListDeployments(serverID, site.ID)
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
			{Header: "Commit", Width: 10},
			{Header: "Message", Width: 32},
			{Header: "Created", Width: 20},
		}

		var rows []tui.TableRow
		for _, d := range deployments {
			message := ""
			if d.CommitData != nil {
				message = truncate(d.CommitData.Message, 30)
			}
			rows = append(rows, tui.TableRow{
				Columns: []string{output.StatusDot(d.Status), d.ShortGitHash, message, d.CreatedAt},
			})
		}

		choice, err := tui.SelectFromTable("Select deployment to view logs", columns, rows, "Back")
		if err != nil || choice == len(rows) {
			return
		}

		viewDeploymentLogs(client, cfg, serverID, serverName, site, deployments[choice])
	}
}

func viewDeploymentLogs(client *api.Client, cfg *config.Config, serverID, serverName string, site api.SiteResponse, deployment api.DeploymentResponse) {
	if deployment.TaskID == nil {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Sites", site.Address, "Deployment")
		tui.ShowInfo("No task output available for this deployment")
		tui.WaitForEnter()
		return
	}

	jwt, err := client.ExchangeToken()
	if err != nil {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Sites", site.Address, "Deployment")
		tui.ShowError(fmt.Sprintf("Failed to authenticate: %s", err))
		tui.WaitForEnter()
		return
	}

	ws, err := api.NewLogsWSClient(cfg, jwt, serverID, "task", *deployment.TaskID)
	if err != nil {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Sites", site.Address, "Deployment")
		tui.ShowError(fmt.Sprintf("Failed to connect to log stream: %s", err))
		tui.WaitForEnter()
		return
	}
	defer ws.Close()

	info := logview.Info{
		Title:  fmt.Sprintf("Deployment: %s", site.Address),
		Status: deployment.Status,
		Commit: deployment.ShortGitHash,
	}

	info.Lines = append(info.Lines, struct{ Label, Value string }{"Commit", deployment.ShortGitHash})
	if deployment.CommitData != nil && deployment.CommitData.Message != "" {
		msg := deployment.CommitData.Message
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}
		info.Lines = append(info.Lines, struct{ Label, Value string }{"Message", msg})
	}
	info.Lines = append(info.Lines, struct{ Label, Value string }{"Created", deployment.CreatedAt})

	if err := logview.Run(info, ws); err != nil {
		tui.ShowError(fmt.Sprintf("Log viewer error: %s", err))
		tui.WaitForEnter()
	}
}

func viewEnvFile(client *api.Client, serverID, siteID, serverName, siteName string) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Sites", siteName, "Environment")

	files, err := client.ListFiles(serverID, siteID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list files: %s", err))
		tui.WaitForEnter()
		return
	}

	var envFile *api.FileOnServer
	for _, f := range files {
		if f.Type == "environment" {
			envFile = &f
			break
		}
	}

	if envFile == nil {
		tui.ShowInfo("No environment file found for this site")
		tui.WaitForEnter()
		return
	}

	content, err := client.GetFileContent(serverID, siteID, envFile.ShowRoute)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to get environment file: %s", err))
		tui.WaitForEnter()
		return
	}

	fmt.Println(tui.CreateBox("Environment File", content.Content, 80))
	fmt.Println()
	fmt.Println(tui.Dim.Render("To update, use: lctl env push --server <id> --site <id> --file .env"))
	fmt.Println()
	tui.WaitForEnter()
}

func viewSiteLogs(client *api.Client, serverID, siteID, serverName, siteName string) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Sites", siteName, "Logs")

		files, err := client.ListFiles(serverID, siteID)
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to list files: %s", err))
			tui.WaitForEnter()
			return
		}

		var logFiles []api.FileOnServer
		for _, f := range files {
			if f.FileType == "log" || strings.Contains(strings.ToLower(f.Name), "log") {
				logFiles = append(logFiles, f)
			}
		}

		if len(logFiles) == 0 {
			tui.ShowInfo("No log files found")
			tui.WaitForEnter()
			return
		}

		columns := []tui.TableColumn{
			{Header: "Name", Width: 24},
			{Header: "Description", Width: 30},
			{Header: "Path", Width: 40},
		}

		var rows []tui.TableRow
		for _, f := range logFiles {
			rows = append(rows, tui.TableRow{
				Columns: []string{f.Name, f.Description, f.Path},
			})
		}

		choice, err := tui.SelectFromTable("Select a log to view", columns, rows, "Back")
		if err != nil || choice == len(rows) {
			return
		}

		showSiteLogContent(client, serverID, siteID, serverName, siteName, logFiles[choice])
	}
}

func showSiteLogContent(client *api.Client, serverID, siteID, serverName, siteName string, file api.FileOnServer) {
	content, err := client.GetFileContent(serverID, siteID, file.ShowRoute)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to get log content: %s", err))
		tui.WaitForEnter()
		return
	}

	info := logview.Info{
		Title: fmt.Sprintf("Log: %s", file.Name),
	}
	info.Lines = append(info.Lines,
		struct{ Label, Value string }{"Site", siteName},
		struct{ Label, Value string }{"Path", file.Path},
	)

	if err := logview.RunStatic(info, content.Content); err != nil {
		tui.ShowError(fmt.Sprintf("Log viewer error: %s", err))
		tui.WaitForEnter()
	}
}

func runSiteCommand(client *api.Client, serverID, siteID, serverName, siteName string) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Sites", siteName, "Run Command")

	command, err := tui.GetInput("Enter command to run", "e.g. php artisan migrate:status", false, func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("command cannot be empty")
		}
		return nil
	})
	if err != nil {
		return
	}

	var confirm bool
	huh.NewConfirm().
		Title(fmt.Sprintf("Run %q on %s?", command, siteName)).
		Value(&confirm).
		Run()

	if !confirm {
		fmt.Println(tui.Dim.Render("Cancelled"))
		tui.WaitForEnter()
		return
	}

	result, err := client.CreateCommand(serverID, siteID, api.CreateCommandRequest{
		Command: command,
	})
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to execute command: %s", err))
		tui.WaitForEnter()
		return
	}

	fmt.Print(tui.Dim.Render("Executing... "))

	commandID := result.ID
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		commands, err := client.ListCommands(serverID, siteID)
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to check command status: %s", err))
			tui.WaitForEnter()
			return
		}

		for _, cmd := range commands {
			if cmd.ID == commandID {
				result = &cmd
				break
			}
		}

		if result.Status == "completed" || result.Status == "failed" {
			break
		}
	}

	fmt.Println()
	fmt.Println()

	if result.Output != nil {
		fmt.Println(tui.CreateBox("Output", *result.Output, 80))
	}

	if result.Status == "failed" {
		exitCode := 1
		if result.ExitCode != nil {
			exitCode = *result.ExitCode
		}
		tui.ShowError(fmt.Sprintf("Command failed with exit code %d", exitCode))
	} else {
		tui.ShowSuccess("Command completed successfully")
	}

	fmt.Println()
	tui.WaitForEnter()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
