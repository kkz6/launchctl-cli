package sites

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var createServerIDFlag string

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new site",
	RunE: func(cmd *cobra.Command, args []string) error {
		if createServerIDFlag == "" {
			return fmt.Errorf("--server flag is required")
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		fmt.Println()
		fmt.Println(tui.Title.Render("  Create Site"))
		fmt.Println()

		var address, siteType, phpVersion, webFolder string
		var zeroDowntime bool

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Domain / Address").
					Placeholder("example.com").
					Value(&address),
				huh.NewSelect[string]().
					Title("Site Type").
					Options(
						huh.NewOption("PHP", "php"),
						huh.NewOption("Static / HTML", "static"),
						huh.NewOption("Node.js", "node"),
						huh.NewOption("Proxy", "proxy"),
					).
					Value(&siteType),
				huh.NewInput().
					Title("PHP Version").
					Placeholder("8.2").
					Value(&phpVersion),
				huh.NewInput().
					Title("Web Folder").
					Placeholder("public").
					Value(&webFolder),
				huh.NewConfirm().
					Title("Zero Downtime Deployment?").
					Value(&zeroDowntime),
			),
		).
			WithTheme(tui.FormTheme()).
			WithWidth(60)

		if err := form.Run(); err != nil {
			return err
		}

		site, err := client.CreateSite(createServerIDFlag, api.CreateSiteRequest{
			Address:      address,
			Type:         siteType,
			PHPVersion:   phpVersion,
			WebFolder:    webFolder,
			ZeroDowntime: zeroDowntime,
		})
		if err != nil {
			return fmt.Errorf("failed to create site: %w", err)
		}

		fmt.Println()
		tui.ShowSuccess("Site created successfully")
		fmt.Println(tui.Label.Render("ID:") + tui.Value.Render(site.ID))
		fmt.Println(tui.Label.Render("Address:") + tui.Value.Render(site.Address))
		fmt.Println(tui.Label.Render("Status:") + tui.Value.Render(site.Status))
		fmt.Println()

		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&createServerIDFlag, "server", "", "Server ID (required)")
}
