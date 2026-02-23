package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var (
	runServerFlag  string
	runSiteFlag    string
	runHistoryFlag bool
)

var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Execute a command on a remote site",
	Long:  "Run a command on a remote server/site and display the output. Use --history to list previous commands.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(runServerFlag)
		if err != nil {
			return err
		}

		siteID, err := resolve.SiteID(runSiteFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		jsonOutput, _ := cmd.Flags().GetBool("json")

		if runHistoryFlag {
			return showCommandHistory(client, serverID, siteID, jsonOutput)
		}

		if len(args) == 0 {
			return fmt.Errorf("command argument is required (or use --history)")
		}

		return executeCommand(cmd, client, serverID, siteID, args[0])
	},
}

func showCommandHistory(client *api.Client, serverID, siteID string, jsonOut bool) error {
	commands, err := client.ListCommands(serverID, siteID)
	if err != nil {
		return fmt.Errorf("failed to list commands: %w", err)
	}

	if jsonOut {
		data, _ := json.MarshalIndent(commands, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	var rows [][]string
	for _, c := range commands {
		exitCode := "-"
		if c.ExitCode != nil {
			exitCode = fmt.Sprintf("%d", *c.ExitCode)
		}

		rows = append(rows, []string{
			c.ID,
			truncate(c.Command, 40),
			c.Status,
			exitCode,
			c.CreatedAt,
		})
	}

	output.RenderTable("Command History", []string{"ID", "Command", "Status", "Exit Code", "Created"}, rows)
	return nil
}

func executeCommand(cmd *cobra.Command, client *api.Client, serverID, siteID, command string) error {
	result, err := client.CreateCommand(serverID, siteID, api.CreateCommandRequest{
		Command: command,
	})
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	ciMode, _ := cmd.Flags().GetBool("ci")

	if !ciMode {
		fmt.Print(tui.Dim.Render("Executing... "))
	}

	commandID := result.ID
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		commands, err := client.ListCommands(serverID, siteID)
		if err != nil {
			return fmt.Errorf("failed to check command status: %w", err)
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

	if !ciMode {
		fmt.Println()
	}

	if result.Output != nil {
		fmt.Print(*result.Output)
	}

	if result.Status == "failed" {
		exitCode := 1
		if result.ExitCode != nil {
			exitCode = *result.ExitCode
		}
		if !ciMode {
			tui.ShowError(fmt.Sprintf("Command failed with exit code %d", exitCode))
		}
		os.Exit(exitCode)
	}

	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func init() {
	runCmd.Flags().StringVar(&runServerFlag, "server", "", "Server ID")
	runCmd.Flags().StringVar(&runSiteFlag, "site", "", "Site ID")
	runCmd.Flags().BoolVar(&runHistoryFlag, "history", false, "List previous commands")
}
