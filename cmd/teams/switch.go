package teams

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch active team",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		teams, err := client.ListTeams()
		if err != nil {
			return fmt.Errorf("failed to list teams: %w", err)
		}

		if len(teams) == 0 {
			return fmt.Errorf("no teams found")
		}

		fmt.Println()
		fmt.Println(tui.Title.Render("  Switch Team"))
		fmt.Println()

		options := make([]huh.Option[string], len(teams))
		for i, t := range teams {
			label := t.Name
			if t.ID == cfg.TeamID {
				label += " (current)"
			}
			options[i] = huh.NewOption(label, t.ID)
		}

		var selectedID string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select team").
					Options(options...).
					Value(&selectedID),
			),
		).
			WithTheme(tui.FormTheme()).
			WithWidth(60)

		if err := form.Run(); err != nil {
			return err
		}

		if err := client.SwitchTeam(selectedID); err != nil {
			return fmt.Errorf("failed to switch team: %w", err)
		}

		for _, t := range teams {
			if t.ID == selectedID {
				cfg.TeamID = t.ID
				cfg.TeamName = t.Name
				break
			}
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println()
		tui.ShowSuccess(fmt.Sprintf("Switched to team: %s", cfg.TeamName))
		fmt.Println()
		return nil
	},
}
