package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if jsonOutput {
			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Println(tui.Title.Render("Configuration"))
		fmt.Println()
		fmt.Println(tui.Label.Render("Team:") + tui.Value.Render(displayOrNone(cfg.TeamName)))
		fmt.Println(tui.Label.Render("User:") + tui.Value.Render(displayOrNone(cfg.UserEmail)))
		fmt.Println(tui.Label.Render("Authenticated:") + authStatusText(cfg.IsAuthenticated()))

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  "Available keys: team_id, team_name",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		if err := cfg.Set(key, value); err != nil {
			return err
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Set %s = %s", key, value)))
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}

func displayOrNone(s string) string {
	if s == "" {
		return tui.Dim.Render("not set")
	}
	return s
}

func authStatusText(authenticated bool) string {
	if authenticated {
		return tui.Success.Render("yes")
	}
	return tui.Dim.Render("no")
}
