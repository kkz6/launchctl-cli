package teams

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/spf13/cobra"
)

var membersTeamIDFlag string

var membersCmd = &cobra.Command{
	Use:   "members",
	Short: "List team members",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		teamID := membersTeamIDFlag
		if teamID == "" {
			teamID = cfg.TeamID
		}

		if teamID == "" {
			return fmt.Errorf("no team selected, use --team or switch teams first")
		}

		members, err := client.GetTeamMembers(teamID)
		if err != nil {
			return fmt.Errorf("failed to list members: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(members, "", "  ")
			fmt.Println(string(data))
			return nil
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
		return nil
	},
}

func init() {
	membersCmd.Flags().StringVar(&membersTeamIDFlag, "team", "", "Team ID (default: current team)")
}
