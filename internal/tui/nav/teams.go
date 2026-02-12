package nav

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/tui"
)

func teamsMenu(client *api.Client, cfg *config.Config) {
	for {
		clearScreen()
		printHeader("launchctl", "Teams")

		choice, err := tui.SelectFromList("Teams", []string{
			"List Teams",
			"Switch Team",
			"Team Members",
			"Back",
		})
		if err != nil || choice == 3 {
			return
		}

		switch choice {
		case 0:
			listTeams(client, cfg)
		case 1:
			switchTeam(client, cfg)
		case 2:
			teamMembers(client, cfg)
		}
	}
}

func listTeams(client *api.Client, cfg *config.Config) {
	clearScreen()
	printHeader("launchctl", "Teams", "List")

	teams, err := client.ListTeams()
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list teams: %s", err))
		waitForEnter()
		return
	}

	var rows [][]string
	for _, t := range teams {
		current := ""
		if t.ID == cfg.TeamID {
			current = tui.Success.Render("*")
		}

		teamType := "Team"
		if t.PersonalTeam {
			teamType = "Personal"
		}

		rows = append(rows, []string{
			current,
			t.ID,
			t.Name,
			teamType,
		})
	}

	output.RenderTable("Teams", []string{"", "ID", "Name", "Type"}, rows)
	waitForEnter()
}

func switchTeam(client *api.Client, cfg *config.Config) {
	clearScreen()
	printHeader("launchctl", "Teams", "Switch")

	teams, err := client.ListTeams()
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list teams: %s", err))
		waitForEnter()
		return
	}

	if len(teams) == 0 {
		tui.ShowInfo("No teams found")
		waitForEnter()
		return
	}

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
		return
	}

	if err := client.SwitchTeam(selectedID); err != nil {
		tui.ShowError(fmt.Sprintf("Failed to switch team: %s", err))
		waitForEnter()
		return
	}

	for _, t := range teams {
		if t.ID == selectedID {
			cfg.TeamID = t.ID
			cfg.TeamName = t.Name
			break
		}
	}

	if err := cfg.Save(); err != nil {
		tui.ShowError(fmt.Sprintf("Failed to save config: %s", err))
		waitForEnter()
		return
	}

	fmt.Println()
	tui.ShowSuccess(fmt.Sprintf("Switched to team: %s", cfg.TeamName))
	waitForEnter()
}

func teamMembers(client *api.Client, cfg *config.Config) {
	clearScreen()
	printHeader("launchctl", "Teams", "Members")

	teamID := cfg.TeamID
	if teamID == "" {
		tui.ShowWarning("No team selected. Switch teams first.")
		waitForEnter()
		return
	}

	members, err := client.GetTeamMembers(teamID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list members: %s", err))
		waitForEnter()
		return
	}

	var rows [][]string
	for _, m := range members {
		rows = append(rows, []string{
			m.Name,
			m.Email,
			m.Role,
		})
	}

	output.RenderTable("Team Members", []string{"Name", "Email", "Role"}, rows)
	waitForEnter()
}
