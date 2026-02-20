package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var (
	initServerFlag string
	initSiteFlag   string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project configuration",
	Long:  "Creates a .launchctl.yml in the current directory to bind this project to a server and site.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cfg.IsAuthenticated() {
			return fmt.Errorf("not authenticated — run `lctl login` first")
		}

		client := api.NewClient(cfg)

		serverID := initServerFlag
		siteID := initSiteFlag

		if serverID == "" || siteID == "" {
			if ciMode {
				return fmt.Errorf("--server and --site flags are required in CI mode")
			}

			servers, _, err := client.ListServers()
			if err != nil {
				return fmt.Errorf("failed to list servers: %w", err)
			}

			if len(servers) == 0 {
				return fmt.Errorf("no servers found")
			}

			if serverID == "" {
				var serverOptions []huh.Option[string]
				for _, s := range servers {
					label := s.Name
					if s.PublicIPv4 != nil {
						label += fmt.Sprintf(" (%s)", *s.PublicIPv4)
					}
					serverOptions = append(serverOptions, huh.NewOption(label, s.ID))
				}

				err = huh.NewForm(
					huh.NewGroup(
						huh.NewSelect[string]().
							Title("Select a server").
							Options(serverOptions...).
							Value(&serverID),
					),
				).WithTheme(tui.FormTheme()).Run()
				if err != nil {
					return err
				}
			}

			sites, err := client.ListSites(serverID)
			if err != nil {
				return fmt.Errorf("failed to list sites: %w", err)
			}

			if len(sites) == 0 {
				return fmt.Errorf("no sites found on this server")
			}

			if siteID == "" {
				var siteOptions []huh.Option[string]
				for _, s := range sites {
					siteOptions = append(siteOptions, huh.NewOption(s.Address, s.ID))
				}

				err = huh.NewForm(
					huh.NewGroup(
						huh.NewSelect[string]().
							Title("Select a site").
							Options(siteOptions...).
							Value(&siteID),
					),
				).WithTheme(tui.FormTheme()).Run()
				if err != nil {
					return err
				}
			}
		}

		projectCfg := &config.ProjectConfig{
			Server: serverID,
			Site:   siteID,
		}

		if err := config.SaveProject(projectCfg); err != nil {
			return fmt.Errorf("failed to save project config: %w", err)
		}

		fmt.Println()
		fmt.Println(tui.Success.Render("Project initialized"))
		fmt.Println(tui.Dim.Render("  Created .launchctl.yml"))
		fmt.Println()

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initServerFlag, "server", "", "Server ID")
	initCmd.Flags().StringVar(&initSiteFlag, "site", "", "Site ID")
	rootCmd.AddCommand(initCmd)
}
