package nav

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/tui"
	deploytui "github.com/kkz6/launchctl/internal/tui/deploy"
)

func sitesMenu(client *api.Client, cfg *config.Config, serverID, serverName string) {
	for {
		clearScreen()
		printHeader("launchctl", "Servers", serverName, "Sites")

		sites, err := client.ListSites(serverID)
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to list sites: %s", err))
			waitForEnter()
			return
		}

		if len(sites) == 0 {
			tui.ShowInfo("No sites found on this server")
		} else {
			var rows [][]string
			for _, s := range sites {
				deployStatus := ""
				if s.LatestDeployment != nil {
					deployStatus = s.LatestDeployment.Status
				}
				rows = append(rows, []string{
					s.ID,
					s.Address,
					s.Type,
					output.StatusDot(s.Status),
					deployStatus,
				})
			}
			output.RenderTable("Sites", []string{"ID", "Address", "Type", "Status", "Last Deploy"}, rows)
		}

		options := make([]string, 0, len(sites)+2)
		for _, s := range sites {
			options = append(options, s.Address)
		}
		options = append(options, "Create Site", "Back")

		choice, err := tui.SelectFromList("Select a site", options)
		if err != nil || choice == len(options)-1 {
			return
		}

		if choice == len(options)-2 {
			createSite(client, cfg, serverID, serverName)
			continue
		}

		siteActions(client, cfg, serverID, serverName, sites[choice])
	}
}

func siteActions(client *api.Client, cfg *config.Config, serverID, serverName string, site api.SiteResponse) {
	for {
		clearScreen()
		printHeader("launchctl", "Servers", serverName, "Sites", site.Address)

		choice, err := tui.SelectFromList(
			fmt.Sprintf("Site: %s", site.Address),
			[]string{"Show Details", "Deploy", "View Deployments", "Back"},
		)
		if err != nil || choice == 3 {
			return
		}

		switch choice {
		case 0:
			showSiteDetails(serverName, site)
		case 1:
			deploySite(client, cfg, serverID, serverName, site)
		case 2:
			viewDeployments(client, serverID, serverName, site)
		}
	}
}

func showSiteDetails(serverName string, site api.SiteResponse) {
	clearScreen()
	printHeader("launchctl", "Servers", serverName, "Sites", site.Address, "Details")

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

	waitForEnter()
}

func deploySite(client *api.Client, cfg *config.Config, serverID, serverName string, site api.SiteResponse) {
	clearScreen()
	printHeader("launchctl", "Servers", serverName, "Sites", site.Address, "Deploy")

	var confirm bool
	huh.NewConfirm().
		Title(fmt.Sprintf("Deploy %q?", site.Address)).
		Description("This will trigger a new deployment.").
		Value(&confirm).
		Run()

	if !confirm {
		fmt.Println(tui.Dim.Render("Cancelled"))
		waitForEnter()
		return
	}

	deployment, err := client.DeploySite(serverID, site.ID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to trigger deployment: %s", err))
		waitForEnter()
		return
	}

	tui.ShowSuccess(fmt.Sprintf("Deployment %s triggered", deployment.ID))

	ws, err := api.NewWSClient(cfg)
	if err != nil {
		tui.ShowWarning("Could not connect to live logs, deployment is running in the background")
		waitForEnter()
		return
	}
	defer ws.Close()

	channel := fmt.Sprintf("team:%s", cfg.TeamID)
	if err := ws.Subscribe(channel); err != nil {
		tui.ShowWarning("Could not subscribe to events")
		waitForEnter()
		return
	}

	model := deploytui.NewModel(site.Address, ws, channel)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		tui.ShowError(fmt.Sprintf("Deploy view error: %s", err))
		waitForEnter()
	}
}

func viewDeployments(client *api.Client, serverID, serverName string, site api.SiteResponse) {
	clearScreen()
	printHeader("launchctl", "Servers", serverName, "Sites", site.Address, "Deployments")

	deployments, err := client.ListDeployments(serverID, site.ID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list deployments: %s", err))
		waitForEnter()
		return
	}

	var rows [][]string
	for _, d := range deployments {
		commit := d.ShortGitHash
		message := ""
		if d.CommitData != nil {
			message = truncate(d.CommitData.Message, 40)
		}
		rows = append(rows, []string{
			d.ID,
			output.StatusDot(d.Status),
			commit,
			message,
			d.CreatedAt,
		})
	}

	output.RenderTable("Deployments", []string{"ID", "Status", "Commit", "Message", "Created"}, rows)
	waitForEnter()
}

func createSite(client *api.Client, cfg *config.Config, serverID, serverName string) {
	clearScreen()
	printHeader("launchctl", "Servers", serverName, "Sites", "Create Site")

	var address, siteType, phpVersion, webFolder string
	var zeroDowntime bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Domain / Address").
				Placeholder("example.com").
				Value(&address),
			huh.NewSelect[string]().
				Title("Site Type").
				Options(
					huh.NewOption("PHP", "php"),
					huh.NewOption("Static / HTML", "static"),
					huh.NewOption("Node.js", "node"),
					huh.NewOption("Proxy", "proxy"),
				).
				Value(&siteType),
			huh.NewInput().
				Title("PHP Version").
				Placeholder("8.2").
				Value(&phpVersion),
			huh.NewInput().
				Title("Web Folder").
				Placeholder("public").
				Value(&webFolder),
			huh.NewConfirm().
				Title("Zero Downtime Deployment?").
				Value(&zeroDowntime),
		),
	).
		WithTheme(tui.FormTheme()).
		WithWidth(60)

	if err := form.Run(); err != nil {
		return
	}

	site, err := client.CreateSite(serverID, api.CreateSiteRequest{
		Address:      address,
		Type:         siteType,
		PHPVersion:   phpVersion,
		WebFolder:    webFolder,
		ZeroDowntime: zeroDowntime,
	})
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to create site: %s", err))
		waitForEnter()
		return
	}

	fmt.Println()
	tui.ShowSuccess("Site created successfully")
	fmt.Println(tui.Label.Render("ID:") + tui.Value.Render(site.ID))
	fmt.Println(tui.Label.Render("Address:") + tui.Value.Render(site.Address))
	fmt.Println(tui.Label.Render("Status:") + tui.Value.Render(site.Status))
	waitForEnter()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
