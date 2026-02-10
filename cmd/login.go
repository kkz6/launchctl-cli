package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with an API token",
	Long:  "Authenticate using a personal access token generated from the web dashboard.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var token string

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("API Token").
					Placeholder("Paste your token from the dashboard").
					EchoMode(huh.EchoModePassword).
					Value(&token),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}

		if token == "" {
			return fmt.Errorf("token is required")
		}

		client := api.NewClient(cfg)

		user, err := client.ValidateToken(token)
		if err != nil {
			return fmt.Errorf("invalid token: %w", err)
		}

		cfg.AccessToken = token

		if user.TwoFactorEnabled {
			var code string

			twoFactorForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Two-Factor Code").
						Placeholder("Enter your 6-digit code").
						Value(&code),
				),
			)

			if err := twoFactorForm.Run(); err != nil {
				cfg.AccessToken = ""
				return err
			}

			if code == "" {
				cfg.AccessToken = ""
				return fmt.Errorf("two-factor code is required")
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

		fmt.Println()
		fmt.Println(tui.Success.Render("Logged in successfully"))
		fmt.Println()
		fmt.Println(tui.Label.Render("User:") + tui.Value.Render(user.Name))
		fmt.Println(tui.Label.Render("Email:") + tui.Value.Render(user.Email))
		if cfg.TeamName != "" {
			fmt.Println(tui.Label.Render("Team:") + tui.Value.Render(cfg.TeamName))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
