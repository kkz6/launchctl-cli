package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var (
	initServerFlag      string
	initSiteFlag        string
	initProjectFlag     string
	initApplicationFlag string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project configuration",
	Long:  "Creates a .launchctl.yml in the current directory to bind this directory to a site or Docker project.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cfg.IsAuthenticated() {
			return fmt.Errorf("not authenticated — run `lctl login` first")
		}

		client := api.NewClient(cfg)

		nonInteractive := ciMode || jsonOutput
		if err := validateInitTargetFlags(initSiteFlag, initProjectFlag, initApplicationFlag); err != nil {
			return err
		}

		serverID := initServerFlag
		if serverID == "" {
			if nonInteractive {
				return fmt.Errorf("--server is required in CI or JSON mode")
			}
			servers, _, err := client.ListServers()
			if err != nil {
				return fmt.Errorf("failed to list servers: %w", err)
			}
			if len(servers) == 0 {
				return fmt.Errorf("no servers found")
			}
			var options []huh.Option[string]
			for _, server := range servers {
				label := server.Name
				if server.PublicIPv4 != nil {
					label += fmt.Sprintf(" (%s)", *server.PublicIPv4)
				}
				options = append(options, huh.NewOption(label, server.ID))
			}
			if err := runSelect("Select a server", options, &serverID); err != nil {
				return err
			}
		}

		server, err := client.GetServer(serverID)
		if err != nil {
			return fmt.Errorf("failed to get server: %w", err)
		}

		projectCfg := &config.ProjectConfig{Server: serverID}
		if server.Type == "docker" {
			if initSiteFlag != "" {
				return fmt.Errorf("--site cannot be used with a Docker server; use --project")
			}

			projectID := initProjectFlag
			if projectID == "" {
				if nonInteractive {
					return fmt.Errorf("--project is required for a Docker server in CI or JSON mode")
				}
				projects, err := client.ListDockerProjects(serverID)
				if err != nil {
					return fmt.Errorf("failed to list Docker projects: %w", err)
				}
				if len(projects) == 0 {
					return fmt.Errorf("no Docker projects found on this server; create one with `lctl docker projects create`")
				}
				options := make([]huh.Option[string], 0, len(projects))
				for _, project := range projects {
					options = append(options, huh.NewOption(project.Name, project.ID))
				}
				if err := runSelect("Select a Docker project", options, &projectID); err != nil {
					return err
				}
			} else if _, err := client.GetDockerProject(serverID, projectID); err != nil {
				return fmt.Errorf("failed to get Docker project: %w", err)
			}

			applicationID := initApplicationFlag
			if applicationID != "" {
				if _, err := client.GetDockerApplication(serverID, projectID, applicationID); err != nil {
					return fmt.Errorf("failed to get Docker application: %w", err)
				}
			} else if !nonInteractive {
				applications, err := client.ListDockerApplications(serverID, projectID)
				if err != nil {
					return fmt.Errorf("failed to list Docker applications: %w", err)
				}
				if len(applications) > 0 {
					options := []huh.Option[string]{huh.NewOption("Project only", "")}
					for _, application := range applications {
						options = append(options, huh.NewOption(application.Name, application.ID))
					}
					if err := runSelect("Select a Docker application (optional)", options, &applicationID); err != nil {
						return err
					}
				}
			}

			projectCfg.DockerProject = projectID
			projectCfg.DockerApplication = applicationID
		} else {
			if initProjectFlag != "" || initApplicationFlag != "" {
				return fmt.Errorf("--project and --application require a Docker server; use --site for this server")
			}

			siteID := initSiteFlag
			if siteID == "" {
				if nonInteractive {
					return fmt.Errorf("--site is required for a non-Docker server in CI or JSON mode")
				}
				sites, err := client.ListSites(serverID)
				if err != nil {
					return fmt.Errorf("failed to list sites: %w", err)
				}
				if len(sites) == 0 {
					return fmt.Errorf("no sites found on this server")
				}
				options := make([]huh.Option[string], 0, len(sites))
				for _, site := range sites {
					options = append(options, huh.NewOption(site.Address, site.ID))
				}
				if err := runSelect("Select a site", options, &siteID); err != nil {
					return err
				}
			} else if _, err := client.GetSite(serverID, siteID); err != nil {
				return fmt.Errorf("failed to get site: %w", err)
			}
			projectCfg.Site = siteID
		}

		if err := config.SaveProject(projectCfg); err != nil {
			return fmt.Errorf("failed to save project config: %w", err)
		}

		if jsonOutput {
			data, err := json.MarshalIndent(projectCfg, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		} else {
			fmt.Println()
			fmt.Println(tui.Success.Render("Project initialized"))
			fmt.Println(tui.Dim.Render("  Created .launchctl.yml"))
			fmt.Println()
		}

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initServerFlag, "server", "", "Server ID")
	initCmd.Flags().StringVar(&initSiteFlag, "site", "", "Site ID")
	initCmd.Flags().StringVar(&initProjectFlag, "project", "", "Docker project ID")
	initCmd.Flags().StringVar(&initApplicationFlag, "application", "", "Docker application ID (optional)")
	rootCmd.AddCommand(initCmd)
}

func validateInitTargetFlags(siteID, projectID, applicationID string) error {
	if siteID != "" && (projectID != "" || applicationID != "") {
		return fmt.Errorf("--site cannot be combined with --project or --application")
	}
	if applicationID != "" && projectID == "" {
		return fmt.Errorf("--application requires --project")
	}
	return nil
}

func runSelect(title string, options []huh.Option[string], value *string) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Options(options...).
				Value(value),
		),
	).WithTheme(tui.FormTheme()).Run()
}
