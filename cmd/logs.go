package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var (
	logsServerFlag string
	logsSiteFlag   string
	logsTypeFlag   string
	logsFollowFlag bool
	logsLinesFlag  int
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View server and site logs",
	Long:  "List available logs or tail a specific log type. Use --type to select a log and --follow to stream updates.",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(logsServerFlag)
		if err != nil {
			return err
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		jsonOutput, _ := cmd.Flags().GetBool("json")

		if logsSiteFlag != "" {
			return handleSiteLogs(client, serverID, logsSiteFlag, jsonOutput)
		}

		return handleServerLogs(client, serverID, jsonOutput)
	},
}

func handleServerLogs(client *api.Client, serverID string, jsonOut bool) error {
	logs, err := client.ListServerLogs(serverID)
	if err != nil {
		return fmt.Errorf("failed to list logs: %w", err)
	}

	if logsTypeFlag == "" {
		return listServerLogs(logs, jsonOut)
	}

	log := findServerLog(logs, logsTypeFlag)
	if log == nil {
		return fmt.Errorf("log type %q not found — run without --type to see available logs", logsTypeFlag)
	}

	return tailServerLog(client, serverID, log)
}

func handleSiteLogs(client *api.Client, serverID, siteID string, jsonOut bool) error {
	files, err := client.ListSiteLogs(serverID, siteID)
	if err != nil {
		return fmt.Errorf("failed to list site logs: %w", err)
	}

	var logFiles []api.FileOnServer
	for _, f := range files {
		if f.FileType == "log" || strings.Contains(strings.ToLower(f.Name), "log") {
			logFiles = append(logFiles, f)
		}
	}

	if logsTypeFlag == "" {
		return listSiteLogs(logFiles, jsonOut)
	}

	var found *api.FileOnServer
	for _, f := range logFiles {
		if strings.EqualFold(f.Name, logsTypeFlag) || strings.Contains(strings.ToLower(f.Name), strings.ToLower(logsTypeFlag)) {
			found = &f
			break
		}
	}

	if found == nil {
		return fmt.Errorf("log type %q not found — run without --type to see available logs", logsTypeFlag)
	}

	return tailSiteLog(client, serverID, siteID, found)
}

func listServerLogs(logs []api.LogInfo, jsonOut bool) error {
	if jsonOut {
		data, _ := json.MarshalIndent(logs, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	var rows [][]string
	for _, l := range logs {
		rows = append(rows, []string{l.Name, l.Software, l.Path})
	}

	output.RenderTable("Server Logs", []string{"Name", "Software", "Path"}, rows)
	return nil
}

func listSiteLogs(logs []api.FileOnServer, jsonOut bool) error {
	if jsonOut {
		data, _ := json.MarshalIndent(logs, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	var rows [][]string
	for _, l := range logs {
		rows = append(rows, []string{l.Name, l.Description, l.Path})
	}

	output.RenderTable("Site Logs", []string{"Name", "Description", "Path"}, rows)
	return nil
}

func findServerLog(logs []api.LogInfo, logType string) *api.LogInfo {
	for _, l := range logs {
		if strings.EqualFold(l.Name, logType) || strings.Contains(strings.ToLower(l.Name), strings.ToLower(logType)) {
			return &l
		}
	}
	return nil
}

func tailServerLog(client *api.Client, serverID string, log *api.LogInfo) error {
	content, err := client.GetServerLogContent(serverID, log.ShowRoute)
	if err != nil {
		return fmt.Errorf("failed to get log content: %w", err)
	}

	printColoredLog(content.Content, logsLinesFlag)

	if !logsFollowFlag {
		return nil
	}

	lastLen := len(content.Content)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		content, err = client.GetServerLogContent(serverID, log.ShowRoute)
		if err != nil {
			continue
		}

		if len(content.Content) > lastLen {
			newContent := content.Content[lastLen:]
			printColoredLog(newContent, 0)
			lastLen = len(content.Content)
		}
	}

	return nil
}

func tailSiteLog(client *api.Client, serverID, siteID string, file *api.FileOnServer) error {
	content, err := client.GetFileContent(serverID, siteID, file.ShowRoute)
	if err != nil {
		return fmt.Errorf("failed to get log content: %w", err)
	}

	printColoredLog(content.Content, logsLinesFlag)

	if !logsFollowFlag {
		return nil
	}

	lastLen := len(content.Content)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		content, err = client.GetFileContent(serverID, siteID, file.ShowRoute)
		if err != nil {
			continue
		}

		if len(content.Content) > lastLen {
			newContent := content.Content[lastLen:]
			printColoredLog(newContent, 0)
			lastLen = len(content.Content)
		}
	}

	return nil
}

func printColoredLog(content string, tailLines int) {
	lines := strings.Split(content, "\n")

	if tailLines > 0 && len(lines) > tailLines {
		lines = lines[len(lines)-tailLines:]
	}

	for _, line := range lines {
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "error") || strings.Contains(lower, "fatal") || strings.Contains(lower, "panic"):
			fmt.Println(tui.Error.Render(line))
		case strings.Contains(lower, "warn"):
			fmt.Println(tui.Warning.Render(line))
		default:
			fmt.Println(line)
		}
	}
}

func init() {
	logsCmd.Flags().StringVar(&logsServerFlag, "server", "", "Server ID")
	logsCmd.Flags().StringVar(&logsSiteFlag, "site", "", "Site ID (for site-specific logs)")
	logsCmd.Flags().StringVarP(&logsTypeFlag, "type", "t", "", "Log type to view (e.g., nginx, laravel)")
	logsCmd.Flags().BoolVarP(&logsFollowFlag, "follow", "f", false, "Follow log output (poll every 2s)")
	logsCmd.Flags().IntVarP(&logsLinesFlag, "lines", "n", 50, "Number of lines to show from tail")
}
