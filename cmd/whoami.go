package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user and team info",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cfg.IsAuthenticated() {
			return fmt.Errorf("not authenticated — run `lctl login` first")
		}

		user := &api.UserResponse{
			Name:  cfg.UserName,
			Email: cfg.UserEmail,
		}
		teamName := cfg.TeamName

		client := api.NewClient(cfg)
		if fetched, err := client.GetUser(); err == nil {
			user = fetched
			if fetched.CurrentTeam != nil {
				teamName = fetched.CurrentTeam.Name
			}
		}

		if jsonOutput {
			out := map[string]string{
				"user_id": user.ID,
				"name":    user.Name,
				"email":   user.Email,
				"team":    teamName,
				"team_id": cfg.TeamID,
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		labelStyle := lipgloss.NewStyle().Foreground(tui.Slate).Width(10)
		valueStyle := lipgloss.NewStyle().Foreground(tui.White)

		var content string
		content += labelStyle.Render("User") + valueStyle.Render(fmt.Sprintf("%s (%s)", user.Name, user.Email)) + "\n"
		content += labelStyle.Render("Team") + valueStyle.Render(teamName)

		fmt.Println()
		fmt.Println(tui.BoxStyle.Width(60).Render(content))
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
