package nav

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/notify"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/terminal"
	"github.com/kkz6/launchctl/internal/tui"
	metricstui "github.com/kkz6/launchctl/internal/tui/metrics"
)

func Run(client *api.Client, cfg *config.Config) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("launchctl")

		favs := cfg.Favorites
		options := make([]string, 0, len(favs)+3)
		for _, f := range favs {
			options = append(options, fmt.Sprintf("★ %s (%s)", f.SiteAddress, f.ServerName))
		}
		options = append(options, "Servers", "Domains (Sites)", "Teams")

		choice, err := tui.SelectFromList("Main Menu", options, "Exit")
		if err != nil || choice == len(options) {
			tui.ClearScreen()
			return
		}

		if choice < len(favs) {
			favoriteActions(client, cfg, favs[choice])
			continue
		}

		switch choice - len(favs) {
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
			tui.WaitForEnter()
			return
		}

		columns := []tui.TableColumn{
			{Header: "Name", Width: 20},
			{Header: "Provider", Width: 14},
			{Header: "Status", Width: 16},
			{Header: "IP", Width: 16},
		}

		var rows []tui.TableRow
		for _, s := range servers {
			ip := ""
			if s.PublicIPv4 != nil {
				ip = *s.PublicIPv4
			}
			rows = append(rows, tui.TableRow{
				Columns: []string{s.Name, s.ProviderLabel, output.StatusDot(s.Status), ip},
			})
		}

		choice, err := tui.SelectFromTable("Select a server", columns, rows, "Back")
		if err != nil || choice == len(rows) {
			return
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
			showServerMetrics(client, cfg, server)
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

func showServerMetrics(client *api.Client, cfg *config.Config, server api.ServerResponse) {
	if !server.Connected {
		tui.ShowError("Server is not connected")
		tui.WaitForEnter()
		return
	}

	jwt, err := client.ExchangeToken()
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to authenticate: %s", err))
		tui.WaitForEnter()
		return
	}

	ws, err := api.NewMetricsWSClient(cfg, jwt, server.ID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to connect to metrics stream: %s", err))
		tui.WaitForEnter()
		return
	}
	defer ws.Close()

	if err := metricstui.Run(server.Name, ws); err != nil {
		tui.ShowError(fmt.Sprintf("Metrics view error: %s", err))
		tui.WaitForEnter()
	}
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
	notify.Success(fmt.Sprintf("Server %q is rebooting", server.Name))
	tui.WaitForEnter()
}

func sshIntoServer(cfg *config.Config, server api.ServerResponse) {
	if !server.Connected {
		tui.ShowError("Server is not connected")
		tui.WaitForEnter()
		return
	}

	client := api.NewClient(cfg)
	jwt, err := client.ExchangeToken()
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to authenticate: %s", err))
		tui.WaitForEnter()
		return
	}

	ip := ""
	if server.PublicIPv4 != nil {
		ip = *server.PublicIPv4
	}

	err = terminal.Connect(cfg, terminal.Options{
		ServerID:   server.ID,
		Username:   server.Username,
		Token:      jwt,
		ServerName: server.Name,
		ServerIP:   ip,
	})

	if err != nil {
		fmt.Println()
		tui.ShowError(fmt.Sprintf("Terminal session ended: %s", err))
		tui.WaitForEnter()
	}
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
