package nav

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/tui"
)

func teamsMenu(client *api.Client, cfg *config.Config) {
	for {
		tui.ClearScreen()
		tui.PrintHeader("lctl", "Teams")

		choice, err := tui.SelectFromList("Teams", []string{
			"List Teams",
			"Switch Team",
			"Team Members",
		}, "Back")
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
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Teams", "List")

	teams, err := client.ListTeams()
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list teams: %s", err))
		tui.WaitForEnter()
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
	tui.WaitForEnter()
}

func switchTeam(client *api.Client, cfg *config.Config) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Teams", "Switch")

	teams, err := client.ListTeams()
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list teams: %s", err))
		tui.WaitForEnter()
		return
	}

	if len(teams) == 0 {
		tui.ShowInfo("No teams found")
		tui.WaitForEnter()
		return
	}

	options := make([]string, 0, len(teams))
	for _, t := range teams {
		label := t.Name
		if t.ID == cfg.TeamID {
			label += " (current)"
		}
		options = append(options, label)
	}

	choice, err := tui.SelectFromList("Select team", options, "Back")
	if err != nil || choice == len(options) {
		return
	}

	selectedID := teams[choice].ID
	if err := client.SwitchTeam(selectedID); err != nil {
		tui.ShowError(fmt.Sprintf("Failed to switch team: %s", err))
		tui.WaitForEnter()
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
		tui.WaitForEnter()
		return
	}

	fmt.Println()
	tui.ShowSuccess(fmt.Sprintf("Switched to team: %s", cfg.TeamName))
	tui.WaitForEnter()
}

func teamMembers(client *api.Client, cfg *config.Config) {
	tui.ClearScreen()
	tui.PrintHeader("lctl", "Teams", "Members")

	teamID := cfg.TeamID
	if teamID == "" {
		tui.ShowWarning("No team selected. Switch teams first.")
		tui.WaitForEnter()
		return
	}

	members, err := client.GetTeamMembers(teamID)
	if err != nil {
		tui.ShowError(fmt.Sprintf("Failed to list members: %s", err))
		tui.WaitForEnter()
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
	tui.WaitForEnter()
}
