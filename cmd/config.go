package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
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
		fmt.Println(tui.Label.Render("Profile:") + tui.Value.Render(cfg.ActiveProfileName()))
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

var profilesCmd = &cobra.Command{
	Use:     "profiles",
	Aliases: []string{"profile"},
	Short:   "Manage configuration profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles := cfg.ListProfiles()
		if len(profiles) == 0 {
			fmt.Println(tui.Dim.Render("No profiles configured"))
			return nil
		}

		sort.Strings(profiles)
		active := cfg.ActiveProfileName()

		fmt.Println(tui.Title.Render("Profiles"))
		fmt.Println()

		nameStyle := lipgloss.NewStyle().Foreground(tui.White).Width(20)
		activeMarker := lipgloss.NewStyle().Foreground(tui.Green).Bold(true)

		for _, name := range profiles {
			p := cfg.Profiles[name]
			line := "  "
			if name == active {
				line += activeMarker.Render("* ")
			} else {
				line += "  "
			}

			line += nameStyle.Render(name)

			if p.UserEmail != "" {
				line += tui.Dim.Render(p.UserEmail)
			}
			if p.TeamName != "" {
				line += tui.Dim.Render(fmt.Sprintf(" (%s)", p.TeamName))
			}

			fmt.Println(line)
		}

		fmt.Println()
		return nil
	},
}

var profilesAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new profile",
	Long:  "Add a new profile and authenticate it. Triggers the login flow.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if cfg.Profiles != nil {
			if _, exists := cfg.Profiles[name]; exists {
				return fmt.Errorf("profile %q already exists", name)
			}
		}

		fmt.Println()
		fmt.Println(tui.Title.Render(fmt.Sprintf("  Add profile: %s", name)))
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

		profile := &config.Profile{
			AccessToken: token,
			UserID:      user.ID,
			UserName:    user.Name,
			UserEmail:   user.Email,
		}

		if user.CurrentTeam != nil {
			profile.TeamID = user.CurrentTeam.ID
			profile.TeamName = user.CurrentTeam.Name
		} else if user.CurrentTeamID != nil {
			profile.TeamID = *user.CurrentTeamID
		}

		if err := cfg.AddProfile(name, profile); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}

		fmt.Println()
		fmt.Println(tui.Success.Render(fmt.Sprintf("Profile %q added", name)))
		fmt.Println(tui.Dim.Render(fmt.Sprintf("  Switch to it with: lctl config profiles use %s", name)))
		fmt.Println()

		return nil
	},
}

var profilesUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch to a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := cfg.UseProfile(name); err != nil {
			return err
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Switched to profile %q", name)))
		return nil
	},
}

var profilesRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := cfg.RemoveProfile(name); err != nil {
			return err
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Profile %q removed", name)))
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)

	profilesCmd.AddCommand(profilesAddCmd)
	profilesCmd.AddCommand(profilesUseCmd)
	profilesCmd.AddCommand(profilesRemoveCmd)
	configCmd.AddCommand(profilesCmd)

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
