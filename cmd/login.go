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
	Short: "Authenticate with the API",
	RunE: func(cmd *cobra.Command, args []string) error {
		var email, password string

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Email").
					Placeholder("you@example.com").
					Value(&email),
				huh.NewInput().
					Title("Password").
					EchoMode(huh.EchoModePassword).
					Value(&password),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}

		client := api.NewClient(cfg)

		auth, err := client.Login(email, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		cfg.AccessToken = auth.AccessToken
		cfg.RefreshToken = auth.RefreshToken
		cfg.UserID = auth.User.ID
		cfg.UserName = auth.User.Name
		cfg.UserEmail = auth.User.Email

		if auth.User.CurrentTeam != nil {
			cfg.TeamID = auth.User.CurrentTeam.ID
			cfg.TeamName = auth.User.CurrentTeam.Name
		} else if auth.User.CurrentTeamID != nil {
			cfg.TeamID = *auth.User.CurrentTeamID
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		fmt.Println()
		fmt.Println(tui.Success.Render("Logged in successfully"))
		fmt.Println()
		fmt.Println(tui.Label.Render("User:") + tui.Value.Render(auth.User.Name))
		fmt.Println(tui.Label.Render("Email:") + tui.Value.Render(auth.User.Email))
		if cfg.TeamName != "" {
			fmt.Println(tui.Label.Render("Team:") + tui.Value.Render(cfg.TeamName))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
