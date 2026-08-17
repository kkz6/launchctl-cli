package nav

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/notify"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/splash"
	"github.com/kkz6/launchctl/internal/terminal"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/kkz6/launchctl/internal/tui/logview"
	metricstui "github.com/kkz6/launchctl/internal/tui/metrics"
)

func Run(client *api.Client, cfg *config.Config, version string, updateVersion func() string) {
	for {
		tui.ClearScreen()
		headerOptions := splash.TerminalOptions(os.Stdout)
		if updateVersion != nil {
			headerOptions.UpdateVersion = updateVersion()
		}
		tui.PrintBrandHeader(splash.Render(version, headerOptions))

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
			{Header: "Type", Width: 12},
			{Header: "Status", Width: 16},
			{Header: "IP", Width: 16},
			{Header: "Sites", Width: 6},
		}

		var rows []tui.TableRow
		for _, s := range servers {
			ip := ""
			if s.PublicIPv4 != nil {
				ip = *s.PublicIPv4
			}
			rows = append(rows, tui.TableRow{
				Columns: []string{s.Name, s.ProviderLabel, s.TypeLabel, output.StatusDot(s.Status), ip, fmt.Sprintf("%d", s.SitesCount)},
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
			[]string{"Show Details", "View Sites", "View Logs", "Services", "Databases", "Firewall", "Cron Jobs", "Daemons", "Metrics", "Reboot", "SSH"},
			"Back",
		)
		if err != nil || choice == 11 {
			return
		}

		switch choice {
		case 0:
			showServerDetails(server)
		case 1:
			sitesMenu(client, cfg, server.ID, server.Name)
		case 2:
			viewServerLogs(client, server.ID, server.Name)
		case 3:
			viewServices(client, server.ID, server.Name)
		case 4:
			viewDatabases(client, server.ID, server.Name)
		case 5:
			viewFirewallRules(client, server.ID, server.Name)
		case 6:
			viewCronJobs(client, server.ID, server.Name)
		case 7:
			viewDaemonsTUI(client, server.ID, server.Name)
		case 8:
			showServerMetrics(client, cfg, server)
		case 9:
			rebootServer(client, server)
		case 10:
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

func viewServerLogs(client *api.Client, serverID, serverName string) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Logs")

		logs, err := client.ListServerLogs(serverID)
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to list logs: %s", err))
			tui.WaitForEnter()
			return
		}

		if len(logs) == 0 {
			tui.ShowInfo("No logs available")
			tui.WaitForEnter()
			return
		}

		columns := []tui.TableColumn{
			{Header: "Name", Width: 20},
			{Header: "Software", Width: 16},
			{Header: "Path", Width: 40},
		}

		var rows []tui.TableRow
		for _, l := range logs {
			rows = append(rows, tui.TableRow{
				Columns: []string{l.Name, l.Software, l.Path},
			})
		}

		choice, err := tui.SelectFromTable("Select a log to view", columns, rows, "Back")
		if err != nil || choice == len(rows) {
			return
		}

		showServerLogContent(client, serverID, serverName, logs[choice])
	}
}

func showServerLogContent(client *api.Client, serverID, serverName string, log api.LogInfo) {
	content, err := client.GetServerLogContent(serverID, log.ShowRoute)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to get log content: %s", err))
		tui.WaitForEnter()
		return
	}

	info := logview.Info{
		Title: fmt.Sprintf("Log: %s", log.Name),
	}
	info.Lines = append(info.Lines,
		struct{ Label, Value string }{"Software", log.Software},
		struct{ Label, Value string }{"Path", log.Path},
	)

	if err := logview.RunStatic(info, content.Content); err != nil {
		tui.ShowError(fmt.Sprintf("Log viewer error: %s", err))
		tui.WaitForEnter()
	}
}

func viewServices(client *api.Client, serverID, serverName string) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Services")

		services, err := client.ListServices(serverID)
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to list services: %s", err))
			tui.WaitForEnter()
			return
		}

		if len(services) == 0 {
			tui.ShowInfo("No services found")
			tui.WaitForEnter()
			return
		}

		columns := []tui.TableColumn{
			{Header: "Name", Width: 20},
			{Header: "Type", Width: 14},
			{Header: "Version", Width: 10},
			{Header: "Status", Width: 16},
		}

		var rows []tui.TableRow
		for _, s := range services {
			version := ""
			if s.Version != nil {
				version = *s.Version
			}
			rows = append(rows, tui.TableRow{
				Columns: []string{s.Name, s.TypeLabel, version, output.StatusDot(s.Status)},
			})
		}

		choice, err := tui.SelectFromTable("Select a service", columns, rows, "Back")
		if err != nil || choice == len(rows) {
			return
		}

		serviceOperationMenu(client, serverID, serverName, services[choice])
	}
}

func serviceOperationMenu(client *api.Client, serverID, serverName string, service api.ServiceResponse) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Services", service.Name)

	choice, err := tui.SelectFromList(
		fmt.Sprintf("Service: %s (%s)", service.Name, output.StatusDot(service.Status)),
		[]string{"Restart", "Start", "Stop"},
		"Back",
	)
	if err != nil || choice == 3 {
		return
	}

	operations := []string{"restart", "start", "stop"}
	op := operations[choice]

	err = client.ServiceOperation(serverID, service.ID, api.ServiceOperationRequest{
		Operation: op,
	})
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to %s service: %s", op, err))
		tui.WaitForEnter()
		return
	}

	tui.ShowSuccess(fmt.Sprintf("Service %s %s initiated", service.Name, op))
	tui.WaitForEnter()
}

func viewDatabases(client *api.Client, serverID, serverName string) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Databases")

	databases, err := client.ListDatabases(serverID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list databases: %s", err))
		tui.WaitForEnter()
		return
	}

	if len(databases) == 0 {
		tui.ShowInfo("No databases found")
		tui.WaitForEnter()
		return
	}

	columns := []tui.TableColumn{
		{Header: "Name", Width: 24},
		{Header: "Status", Width: 16},
		{Header: "Users", Width: 30},
	}

	var rows []tui.TableRow
	for _, db := range databases {
		var userNames []string
		for _, u := range db.Users {
			userNames = append(userNames, u.Name)
		}
		users := ""
		if len(userNames) > 0 {
			users = strings.Join(userNames, ", ")
		}
		rows = append(rows, tui.TableRow{
			Columns: []string{db.Name, output.StatusDot(db.Status), users},
		})
	}

	tui.SelectFromTable("Databases", columns, rows, "Back")
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

func viewFirewallRules(client *api.Client, serverID, serverName string) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Firewall")

	rules, err := client.ListFirewallRules(serverID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list firewall rules: %s", err))
		tui.WaitForEnter()
		return
	}

	if len(rules) == 0 {
		tui.ShowInfo("No firewall rules found")
		tui.WaitForEnter()
		return
	}

	columns := []tui.TableColumn{
		{Header: "Name", Width: 20},
		{Header: "Action", Width: 10},
		{Header: "Port", Width: 10},
		{Header: "From IP", Width: 18},
		{Header: "Status", Width: 16},
	}

	var rows []tui.TableRow
	for _, r := range rules {
		port := ""
		if r.Port != nil {
			port = *r.Port
		}
		fromIP := ""
		if r.FromIPv4 != nil {
			fromIP = *r.FromIPv4
		}

		status := "installed"
		if r.HasFailed {
			status = "failed"
		} else if r.IsPending {
			status = "pending"
		} else if !r.IsInstalled {
			status = "not installed"
		}

		rows = append(rows, tui.TableRow{
			Columns: []string{r.Name, r.ActionLabel, port, fromIP, output.StatusDot(status)},
		})
	}

	tui.SelectFromTable("Firewall Rules", columns, rows, "Back")
}

func viewCronJobs(client *api.Client, serverID, serverName string) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Cron Jobs")

	crons, err := client.ListCrons(serverID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list cron jobs: %s", err))
		tui.WaitForEnter()
		return
	}

	if len(crons) == 0 {
		tui.ShowInfo("No cron jobs found")
		tui.WaitForEnter()
		return
	}

	columns := []tui.TableColumn{
		{Header: "User", Width: 12},
		{Header: "Expression", Width: 16},
		{Header: "Command", Width: 30},
		{Header: "Installed", Width: 12},
	}

	var rows []tui.TableRow
	for _, c := range crons {
		installed := "No"
		if c.IsInstalled {
			installed = "Yes"
		}
		rows = append(rows, tui.TableRow{
			Columns: []string{c.User, c.Expression, c.Command, installed},
		})
	}

	tui.SelectFromTable("Cron Jobs", columns, rows, "Back")
}

func viewDaemonsTUI(client *api.Client, serverID, serverName string) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Servers", serverName, "Daemons")

		daemons, err := client.ListDaemons(serverID)
		if err != nil {
			tui.ShowError(fmt.Sprintf("Failed to list daemons: %s", err))
			tui.WaitForEnter()
			return
		}

		if len(daemons) == 0 {
			tui.ShowInfo("No daemons found")
			tui.WaitForEnter()
			return
		}

		columns := []tui.TableColumn{
			{Header: "Command", Width: 30},
			{Header: "User", Width: 12},
			{Header: "Processes", Width: 10},
			{Header: "Running", Width: 12},
		}

		var rows []tui.TableRow
		for _, d := range daemons {
			status := "stopped"
			if d.Running {
				status = "running"
			}
			rows = append(rows, tui.TableRow{
				Columns: []string{d.Command, d.User, fmt.Sprintf("%d", d.Processes), output.StatusDot(status)},
			})
		}

		choice, err := tui.SelectFromTable("Select a daemon", columns, rows, "Back")
		if err != nil || choice == len(rows) {
			return
		}

		daemonActionMenu(client, serverID, serverName, daemons[choice])
	}
}

func daemonActionMenu(client *api.Client, serverID, serverName string, daemon api.DaemonResponse) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Servers", serverName, "Daemons", daemon.Command)

	status := "stopped"
	if daemon.Running {
		status = "running"
	}

	choice, err := tui.SelectFromList(
		fmt.Sprintf("Daemon: %s (%s)", daemon.Command, output.StatusDot(status)),
		[]string{"Restart"},
		"Back",
	)
	if err != nil || choice == 1 {
		return
	}

	err = client.RestartDaemon(serverID, daemon.ID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to restart daemon: %s", err))
		tui.WaitForEnter()
		return
	}

	tui.ShowSuccess(fmt.Sprintf("Daemon %q restart initiated", daemon.Command))
	tui.WaitForEnter()
}
