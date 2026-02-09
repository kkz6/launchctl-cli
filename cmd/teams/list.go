package teams

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your teams",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		teams, err := client.ListTeams()
		if err != nil {
			return fmt.Errorf("failed to list teams: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(teams, "", "  ")
			fmt.Println(string(data))
			return nil
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
		return nil
	},
}
