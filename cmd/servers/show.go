package servers

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show server details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		server, err := client.GetServer(args[0])
		if err != nil {
			return fmt.Errorf("failed to get server: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(server, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println()
		fmt.Println(tui.Title.Render(server.Name))
		fmt.Println()
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
		fmt.Println(tui.Label.Render("Connected:") + connectedText(server.Connected))

		if server.CPUCores != nil {
			fmt.Println(tui.Label.Render("CPU:") + tui.Value.Render(fmt.Sprintf("%d cores", *server.CPUCores)))
		}
		if server.MemoryInMB != nil {
			fmt.Println(tui.Label.Render("Memory:") + tui.Value.Render(fmt.Sprintf("%d MB", *server.MemoryInMB)))
		}
		if server.StorageInGB != nil {
			fmt.Println(tui.Label.Render("Storage:") + tui.Value.Render(fmt.Sprintf("%d GB", *server.StorageInGB)))
		}

		for _, resource := range serverResourceCounts(*server) {
			fmt.Println(tui.Label.Render(resource.Label+":") + tui.Value.Render(fmt.Sprintf("%d", resource.Count)))
		}
		fmt.Println(tui.Label.Render("Created:") + tui.Value.Render(server.CreatedAt))
		fmt.Println()

		return nil
	},
}

func connectedText(connected bool) string {
	if connected {
		return tui.StatusConnected.Render("yes")
	}
	return tui.StatusDisconnected.Render("no")
}
