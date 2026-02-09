package deploy

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var showServerFlag, showSiteFlag string

var showCmd = &cobra.Command{
	Use:   "show <deployment-id>",
	Short: "Show deployment details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if showServerFlag == "" || showSiteFlag == "" {
			return fmt.Errorf("--server and --site flags are required")
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		d, err := client.GetDeployment(showServerFlag, showSiteFlag, args[0])
		if err != nil {
			return fmt.Errorf("failed to get deployment: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(d, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println()
		fmt.Println(tui.Title.Render("Deployment Details"))
		fmt.Println()
		fmt.Println(tui.Label.Render("ID:") + tui.Value.Render(d.ID))
		fmt.Println(tui.Label.Render("Status:") + output.StatusDot(d.Status))
		fmt.Println(tui.Label.Render("Commit:") + tui.Value.Render(d.ShortGitHash))
		fmt.Println(tui.Label.Render("Rollback:") + tui.Value.Render(fmt.Sprintf("%v", d.IsRollback)))

		if d.CommitData != nil {
			fmt.Println(tui.Label.Render("Message:") + tui.Value.Render(d.CommitData.Message))
			fmt.Println(tui.Label.Render("Author:") + tui.Value.Render(d.CommitData.Author))
		}

		fmt.Println(tui.Label.Render("Created:") + tui.Dim.Render(d.CreatedAt))
		fmt.Println()

		return nil
	},
}

func init() {
	showCmd.Flags().StringVar(&showServerFlag, "server", "", "Server ID (required)")
	showCmd.Flags().StringVar(&showSiteFlag, "site", "", "Site ID (required)")
}
