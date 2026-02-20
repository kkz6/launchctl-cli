package nav

import (
	"fmt"

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
			{Header: "Address", Width: 24},
			{Header: "Type", Width: 10},
			{Header: "Status", Width: 16},
			{Header: "Last Deploy", Width: 12},
		}

		var rows []tui.TableRow
		for _, s := range sites {
			deployStatus := ""
			if s.LatestDeployment != nil {
				deployStatus = s.LatestDeployment.Status
			}
			rows = append(rows, tui.TableRow{
				Columns: []string{s.Address, s.Type, output.StatusDot(s.Status), deployStatus},
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
			[]string{"Show Details", "Deploy", "View Deployments", favLabel},
			"Back",
		)
		if err != nil || choice == 4 {
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
	fmt.Println(tui.Label.Render("Type:") + tui.Value.Render(site.Type))
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

	teamChannel := fmt.Sprintf("team:%s", cfg.TeamID)
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
