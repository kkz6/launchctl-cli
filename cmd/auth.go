package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cfg.IsAuthenticated() {
			fmt.Println()
			fmt.Println(tui.BoxStyle.Width(60).Render(
				tui.Warning.Render("Not authenticated") + "\n\n" +
					tui.Dim.Render("Run ") +
					tui.Bold.Render("lctl login") +
					tui.Dim.Render(" to authenticate."),
			))
			fmt.Println()
			return nil
		}

		client := api.NewClient(cfg)
		user, err := client.GetUser()

		labelStyle := lipgloss.NewStyle().Foreground(tui.Slate).Width(14)
		valueStyle := lipgloss.NewStyle().Foreground(tui.White)

		var content string

		if err != nil {
			content += tui.Error.Render("Token is invalid or expired") + "\n\n"
			content += labelStyle.Render("User") + valueStyle.Render(cfg.UserName) + "\n"
			content += labelStyle.Render("Email") + valueStyle.Render(cfg.UserEmail) + "\n\n"
			content += tui.Dim.Render("Run ") +
				tui.Bold.Render("lctl login") +
				tui.Dim.Render(" to re-authenticate.")
		} else {
			content += tui.Success.Render("Authenticated") + "\n\n"
			content += labelStyle.Render("User") + valueStyle.Render(user.Name) + "\n"
			content += labelStyle.Render("Email") + valueStyle.Render(user.Email)

			if user.TwoFactorEnabled {
				content += "\n" + labelStyle.Render("2FA") + tui.Success.Render("enabled")
			} else {
				content += "\n" + labelStyle.Render("2FA") + tui.Dim.Render("disabled")
			}

			if user.CurrentTeam != nil {
				content += "\n" + labelStyle.Render("Team") + valueStyle.Render(user.CurrentTeam.Name)
			} else if cfg.TeamName != "" {
				content += "\n" + labelStyle.Render("Team") + valueStyle.Render(cfg.TeamName)
			}
		}

		fmt.Println()
		fmt.Println(tui.BoxStyle.Width(60).Render(content))
		fmt.Println()

		return nil
	},
}

func init() {
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
