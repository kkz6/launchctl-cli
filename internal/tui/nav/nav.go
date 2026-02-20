package nav

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/terminal"
	"github.com/kkz6/launchctl/internal/tui"
)

func Run(client *api.Client, cfg *config.Config) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("launchctl")

		choice, err := tui.SelectFromList("Main Menu", []string{
			"Servers",
			"Domains (Sites)",
			"Teams",
		}, "Exit")
		if err != nil || choice == 3 {
			tui.ClearScreen()
			return
		}

		switch choice {
		case 0:
			serversMenu(client, cfg)
		case 1:
			domainsMenu(client, cfg)
		case 2:
			teamsMenu(client, cfg)
		}
	}
}

func serversMenu(client *api.Client, cfg *config.Config) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers")

		servers, _, err := client.ListServers()
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to list servers: %s", err))
			tui.WaitForEnter()
			return
		}

		if len(servers) == 0 {
			tui.ShowInfo("No servers found")
		} else {
			var rows [][]string
			for _, s := range servers {
				ip := ""
				if s.PublicIPv4 != nil {
					ip = *s.PublicIPv4
				}
				rows = append(rows, []string{
					s.ID,
					s.Name,
					s.ProviderLabel,
					output.StatusDot(s.Status),
					ip,
				})
			}
			output.RenderTable("Servers", []string{"ID", "Name", "Provider", "Status", "IP"}, rows)
		}

		options := make([]string, 0, len(servers))
		for _, s := range servers {
			label := s.Name
			if s.PublicIPv4 != nil {
				label += fmt.Sprintf(" (%s)", *s.PublicIPv4)
			}
			options = append(options, label)
		}

		choice, err := tui.SelectFromList("Select a server", options, "Create Server", "Back")
		if err != nil || choice == len(options)+1 {
			return
		}

		if choice == len(options) {
			createServer(client, cfg)
			continue
		}

		serverActions(client, cfg, servers[choice])
	}
}

func serverActions(client *api.Client, cfg *config.Config, server api.ServerResponse) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", server.Name)

		choice, err := tui.SelectFromList(
			fmt.Sprintf("Server: %s", server.Name),
			[]string{"Show Details", "View Sites", "Metrics", "Reboot", "SSH"},
			"Back",
		)
		if err != nil || choice == 5 {
			return
		}

		switch choice {
		case 0:
			showServerDetails(server)
		case 1:
			sitesMenu(client, cfg, server.ID, server.Name)
		case 2:
			showServerMetrics(client, server)
		case 3:
			rebootServer(client, server)
		case 4:
			sshIntoServer(cfg, server)
		}
	}
}

func showServerDetails(server api.ServerResponse) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", server.Name, "Details")

	fmt.Println(tui.Label.Render("ID:") + tui.Value.Render(server.ID))
	fmt.Println(tui.Label.Render("Status:") + output.StatusDot(server.Status))
	fmt.Println(tui.Label.Render("Provider:") + tui.Value.Render(server.ProviderLabel))
	fmt.Println(tui.Label.Render("Type:") + tui.Value.Render(server.TypeLabel))
	fmt.Println(tui.Label.Render("OS:") + tui.Value.Render(server.OperatingSystemLabel))

	if server.PublicIPv4 != nil {
		fmt.Println(tui.Label.Render("IP:") + tui.Value.Render(*server.PublicIPv4))
	}

	fmt.Println(tui.Label.Render("SSH Port:") + tui.Value.Render(fmt.Sprintf("%d", server.SSHPort)))
	fmt.Println(tui.Label.Render("Username:") + tui.Value.Render(server.Username))

	connected := tui.StatusDisconnected.Render("no")
	if server.Connected {
		connected = tui.StatusConnected.Render("yes")
	}
	fmt.Println(tui.Label.Render("Connected:") + connected)

	if server.CPUCores != nil {
		fmt.Println(tui.Label.Render("CPU:") + tui.Value.Render(fmt.Sprintf("%d cores", *server.CPUCores)))
	}
	if server.MemoryInMB != nil {
		fmt.Println(tui.Label.Render("Memory:") + tui.Value.Render(fmt.Sprintf("%d MB", *server.MemoryInMB)))
	}
	if server.StorageInGB != nil {
		fmt.Println(tui.Label.Render("Storage:") + tui.Value.Render(fmt.Sprintf("%d GB", *server.StorageInGB)))
	}

	fmt.Println(tui.Label.Render("Sites:") + tui.Value.Render(fmt.Sprintf("%d", server.SitesCount)))
	fmt.Println(tui.Label.Render("Created:") + tui.Value.Render(server.CreatedAt))

	tui.WaitForEnter()
}

func showServerMetrics(client *api.Client, server api.ServerResponse) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", server.Name, "Metrics")

	metrics, err := client.GetServerMetrics(server.ID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to get metrics: %s", err))
		tui.WaitForEnter()
		return
	}

	fmt.Println(tui.Label.Render("Load:") + tui.Value.Render(fmt.Sprintf("%.2f", metrics.Load)))
	fmt.Println()
	fmt.Println(tui.Label.Render("Memory:") + renderBar(metrics.MemoryUsagePercent))
	fmt.Println(tui.Label.Render("") + tui.Dim.Render(fmt.Sprintf("%.0f MB / %.0f MB (%.1f%%)",
		metrics.MemoryUsed/1024/1024, metrics.MemoryTotal/1024/1024, metrics.MemoryUsagePercent)))
	fmt.Println()
	fmt.Println(tui.Label.Render("Disk:") + renderBar(metrics.DiskUsagePercent))
	fmt.Println(tui.Label.Render("") + tui.Dim.Render(fmt.Sprintf("%.1f GB / %.1f GB (%.1f%%)",
		metrics.DiskUsed/1024/1024/1024, metrics.DiskTotal/1024/1024/1024, metrics.DiskUsagePercent)))
	fmt.Println()
	fmt.Println(tui.Label.Render("Updated:") + tui.Dim.Render(metrics.CreatedAt))

	tui.WaitForEnter()
}

func renderBar(percent float64) string {
	width := 30
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}

	color := tui.Green
	if percent > 80 {
		color = tui.Red
	} else if percent > 60 {
		color = tui.Yellow
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += tui.Bold.Foreground(color).Render("\u2588")
		} else {
			bar += tui.Dim.Render("\u2591")
		}
	}

	return bar + fmt.Sprintf(" %.1f%%", percent)
}

func rebootServer(client *api.Client, server api.ServerResponse) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", server.Name, "Reboot")

	var confirm bool
	huh.NewConfirm().
		Title(fmt.Sprintf("Reboot server %q?", server.Name)).
		Description("This will restart the server and cause brief downtime.").
		Value(&confirm).
		Run()

	if !confirm {
		fmt.Println(tui.Dim.Render("Cancelled"))
		tui.WaitForEnter()
		return
	}

	if err := client.RebootServer(server.ID); err != nil {
		tui.ShowError(fmt.Sprintf("Failed to reboot server: %s", err))
		tui.WaitForEnter()
		return
	}

	tui.ShowSuccess(fmt.Sprintf("Server %q is rebooting", server.Name))
	tui.WaitForEnter()
}

func sshIntoServer(cfg *config.Config, server api.ServerResponse) {
	if !server.Connected {
		tui.ShowError("Server is not connected")
		tui.WaitForEnter()
		return
	}

	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", server.Name, "Terminal")

	client := api.NewClient(cfg)
	jwt, err := client.ExchangeToken()
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to authenticate: %s", err))
		tui.WaitForEnter()
		return
	}

	fmt.Println(tui.Info.Render(fmt.Sprintf("  Connecting to %s...", server.Name)))
	fmt.Println(tui.Dim.Render("  Type 'exit' to return to menu"))
	tui.PrintDivider()
	fmt.Println()

	err = terminal.Connect(cfg, terminal.Options{
		ServerID: server.ID,
		Username: server.Username,
		Token:    jwt,
	})

	fmt.Println()
	if err != nil {
		tui.ShowError(fmt.Sprintf("Terminal session ended: %s", err))
	} else {
		tui.ShowInfo("Terminal session closed")
	}
	tui.WaitForEnter()
}

func createServer(client *api.Client, cfg *config.Config) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", "Create Server")

	opts, err := client.GetCreateServerOptions()
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to fetch options: %s", err))
		tui.WaitForEnter()
		return
	}

	if len(opts.Providers) == 0 {
		tui.ShowWarning("No server providers configured. Add one in the web dashboard first.")
		tui.WaitForEnter()
		return
	}

	var name, providerIdx, serverType, osChoice, region, size string

	providerOptions := make([]huh.Option[string], len(opts.Providers))
	for i, p := range opts.Providers {
		providerOptions[i] = huh.NewOption(p.Name, fmt.Sprintf("%d", i))
	}

	typeOptions := make([]huh.Option[string], len(opts.Types))
	for i, t := range opts.Types {
		typeOptions[i] = huh.NewOption(t.Label, t.Value)
	}

	osOptions := make([]huh.Option[string], len(opts.OperatingSystems))
	for i, o := range opts.OperatingSystems {
		osOptions[i] = huh.NewOption(o.Label, o.Value)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Server Name").
				Placeholder("my-server").
				Value(&name),
			huh.NewSelect[string]().
				Title("Provider").
				Options(providerOptions...).
				Value(&providerIdx),
			huh.NewSelect[string]().
				Title("Type").
				Options(typeOptions...).
				Value(&serverType),
			huh.NewSelect[string]().
				Title("Operating System").
				Options(osOptions...).
				Value(&osChoice),
		),
	).
		WithTheme(tui.FormTheme()).
		WithWidth(60)

	if err := form.Run(); err != nil {
		return
	}

	var idx int
	fmt.Sscanf(providerIdx, "%d", &idx)
	provider := opts.Providers[idx]

	regionOptions := make([]huh.Option[string], len(provider.Regions))
	for i, r := range provider.Regions {
		regionOptions[i] = huh.NewOption(r.Label, r.Value)
	}

	sizeOptions := make([]huh.Option[string], len(provider.Sizes))
	for i, s := range provider.Sizes {
		sizeOptions[i] = huh.NewOption(s.Label, s.Value)
	}

	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Region").
				Options(regionOptions...).
				Value(&region),
			huh.NewSelect[string]().
				Title("Size").
				Options(sizeOptions...).
				Value(&size),
		),
	).
		WithTheme(tui.FormTheme()).
		WithWidth(60)

	if err := form2.Run(); err != nil {
		return
	}

	server, err := client.CreateServer(api.CreateServerRequest{
		Name:            name,
		Provider:        provider.Provider,
		CredentialID:    provider.ID,
		Type:            serverType,
		Region:          region,
		Size:            size,
		OperatingSystem: osChoice,
	})
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to create server: %s", err))
		tui.WaitForEnter()
		return
	}

	fmt.Println()
	tui.ShowSuccess("Server created successfully")
	fmt.Println(tui.Label.Render("ID:") + tui.Value.Render(server.ID))
	fmt.Println(tui.Label.Render("Name:") + tui.Value.Render(server.Name))
	fmt.Println(tui.Label.Render("Status:") + tui.Value.Render(server.StatusLabel))
	tui.WaitForEnter()
}

func domainsMenu(client *api.Client, cfg *config.Config) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Domains")

	servers, _, err := client.ListServers()
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list servers: %s", err))
		tui.WaitForEnter()
		return
	}

	if len(servers) == 0 {
		tui.ShowInfo("No servers found. Create a server first.")
		tui.WaitForEnter()
		return
	}

	options := make([]string, 0, len(servers))
	for _, s := range servers {
		label := s.Name
		if s.PublicIPv4 != nil {
			label += fmt.Sprintf(" (%s)", *s.PublicIPv4)
		}
		options = append(options, label)
	}

	choice, err := tui.SelectFromList("Select a server to view sites", options, "Back")
	if err != nil || choice == len(options) {
		return
	}

	sitesMenu(client, cfg, servers[choice].ID, servers[choice].Name)
}
