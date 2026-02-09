package sites

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var showServerIDFlag string

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show site details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if showServerIDFlag == "" {
			return fmt.Errorf("--server flag is required")
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		site, err := client.GetSite(showServerIDFlag, args[0])
		if err != nil {
			return fmt.Errorf("failed to get site: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(site, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println()
		fmt.Println(tui.Title.Render(site.Address))
		fmt.Println()
		fmt.Println(tui.Label.Render("ID:") + tui.Value.Render(site.ID))
		fmt.Println(tui.Label.Render("Status:") + output.StatusDot(site.Status))
		fmt.Println(tui.Label.Render("Type:") + tui.Value.Render(site.Type))
		fmt.Println(tui.Label.Render("URL:") + tui.Value.Render(site.URL))
		fmt.Println(tui.Label.Render("Path:") + tui.Value.Render(site.Path))
		fmt.Println(tui.Label.Render("PHP:") + tui.Value.Render(site.PHPVersion))
		fmt.Println(tui.Label.Render("Branch:") + tui.Value.Render(site.RepositoryBranch))
		fmt.Println(tui.Label.Render("Zero Downtime:") + tui.Value.Render(fmt.Sprintf("%v", site.ZeroDowntimeDeployment)))
		fmt.Println(tui.Label.Render("Auto Deploy:") + tui.Value.Render(fmt.Sprintf("%v", site.AutoDeployment)))

		if site.RepositoryURL != nil {
			fmt.Println(tui.Label.Render("Repository:") + tui.Value.Render(*site.RepositoryURL))
		}

		if site.LatestDeployment != nil {
			fmt.Println()
			fmt.Println(tui.Subtitle.Render("Latest Deployment"))
			fmt.Println(tui.Label.Render("Status:") + output.StatusDot(site.LatestDeployment.Status))
			fmt.Println(tui.Label.Render("Commit:") + tui.Value.Render(site.LatestDeployment.ShortGitHash))
			if site.LatestDeployment.CommitData != nil {
				fmt.Println(tui.Label.Render("Message:") + tui.Value.Render(site.LatestDeployment.CommitData.Message))
			}
			fmt.Println(tui.Label.Render("Created:") + tui.Dim.Render(site.LatestDeployment.CreatedAt))
		}

		fmt.Println()
		return nil
	},
}

func init() {
	showCmd.Flags().StringVar(&showServerIDFlag, "server", "", "Server ID (required)")
}
