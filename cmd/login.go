package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with an API token",
	Long:  "Authenticate using a personal access token generated from the web dashboard.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println()
		fmt.Println(tui.Title.Render("  Login to lctl"))
		fmt.Println(tui.Dim.Render("  Generate a token at https://launchctl.io/settings/api-tokens"))
		fmt.Println()

		token, err := tui.GetInput(
			"API Token",
			"lctl_xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			true,
			func(s string) error {
				if s == "" {
					return fmt.Errorf("token is required")
				}
				return nil
			},
		)
		if err != nil {
			return err
		}

		client := api.NewClient(cfg)

		user, err := client.ValidateToken(token)
		if err != nil {
			return fmt.Errorf("invalid token: %w", err)
		}

		cfg.AccessToken = token

		if user.TwoFactorEnabled {
			fmt.Println()

			code, err := tui.GetInput(
				"Two-Factor Authentication",
				"000000",
				false,
				func(s string) error {
					if s == "" {
						return fmt.Errorf("two-factor code is required")
					}
					return nil
				},
			)
			if err != nil {
				cfg.AccessToken = ""
				return err
			}

			if err := client.VerifyTwoFactor(code); err != nil {
				cfg.AccessToken = ""
				return fmt.Errorf("two-factor verification failed: %w", err)
			}
		}

		cfg.UserID = user.ID
		cfg.UserName = user.Name
		cfg.UserEmail = user.Email

		if user.CurrentTeam != nil {
			cfg.TeamID = user.CurrentTeam.ID
			cfg.TeamName = user.CurrentTeam.Name
		} else if user.CurrentTeamID != nil {
			cfg.TeamID = *user.CurrentTeamID
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		labelStyle := lipgloss.NewStyle().Foreground(tui.Slate).Width(12)
		valueStyle := lipgloss.NewStyle().Foreground(tui.White)

		var content string
		content += tui.Success.Render("Logged in successfully") + "\n\n"
		content += labelStyle.Render("User") + valueStyle.Render(user.Name) + "\n"
		content += labelStyle.Render("Email") + valueStyle.Render(user.Email)
		if cfg.TeamName != "" {
			content += "\n" + labelStyle.Render("Team") + valueStyle.Render(cfg.TeamName)
		}

		fmt.Println()
		fmt.Println(tui.BoxStyle.Width(60).Render(content))
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
