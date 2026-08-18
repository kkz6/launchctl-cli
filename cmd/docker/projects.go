package docker

import (
	"fmt"
	"strings"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

type projectsOptions struct {
	server string
}

func newProjectsCommand() *cobra.Command {
	opts := &projectsOptions{}
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"project"},
		Short:   "Manage Docker projects",
	}
	cmd.PersistentFlags().StringVar(&opts.server, "server", "", "Docker server ID (or project default)")
	cmd.AddCommand(newProjectsListCommand(opts))
	cmd.AddCommand(newProjectsShowCommand(opts))
	cmd.AddCommand(newProjectsCreateCommand(opts))
	cmd.AddCommand(newProjectsUpdateCommand(opts))
	cmd.AddCommand(newProjectsDeleteCommand(opts))
	return cmd
}

func newProjectsListCommand(opts *projectsOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Docker projects on a server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, serverID, err := dockerClient(opts.server)
			if err != nil {
				return err
			}
			projects, err := client.ListDockerProjects(serverID)
			if err != nil {
				return fmt.Errorf("failed to list Docker projects: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, projects)
			}

			rows := make([][]string, 0, len(projects))
			for _, project := range projects {
				rows = append(rows, []string{
					project.ID,
					project.Name,
					fmt.Sprintf("%d", project.ApplicationsCount),
					fmt.Sprintf("%d", project.ComposesCount),
					fmt.Sprintf("%d", project.DatabasesCount),
					valueOrEmpty(project.Description),
				})
			}
			output.RenderTable("Docker Projects", []string{"ID", "Name", "Apps", "Composes", "Databases", "Description"}, rows)
			return nil
		},
	}
}

func newProjectsShowCommand(opts *projectsOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "show [project-id]",
		Short: "Show Docker project details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID, err := projectIDFromArg(args)
			if err != nil {
				return err
			}
			client, serverID, err := dockerClient(opts.server)
			if err != nil {
				return err
			}
			project, err := client.GetDockerProject(serverID, projectID)
			if err != nil {
				return fmt.Errorf("failed to get Docker project: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, project)
			}

			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), tui.Title.Render(project.Name))
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("ID:")+tui.Value.Render(project.ID))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Server ID:")+tui.Value.Render(project.ServerID))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Description:")+tui.Value.Render(valueOrEmpty(project.Description)))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Applications:")+tui.Value.Render(fmt.Sprintf("%d", project.ApplicationsCount)))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Composes:")+tui.Value.Render(fmt.Sprintf("%d", project.ComposesCount)))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Databases:")+tui.Value.Render(fmt.Sprintf("%d", project.DatabasesCount)))
			fmt.Fprintln(cmd.OutOrStdout(), tui.Label.Render("Created:")+tui.Value.Render(formatTime(project.CreatedAt)))
			fmt.Fprintln(cmd.OutOrStdout())
			return nil
		},
	}
}

func newProjectsCreateCommand(opts *projectsOptions) *cobra.Command {
	var name, description string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Docker project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			client, serverID, err := dockerClient(opts.server)
			if err != nil {
				return err
			}
			project, err := client.CreateDockerProject(serverID, api.CreateDockerProjectRequest{
				Name:        name,
				Description: optionalString(description),
			})
			if err != nil {
				return fmt.Errorf("failed to create Docker project: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, project)
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("Docker project %q created (ID: %s)", project.Name, project.ID)))
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Project description")
	return cmd
}

func newProjectsUpdateCommand(opts *projectsOptions) *cobra.Command {
	var name, description string
	cmd := &cobra.Command{
		Use:   "update [project-id]",
		Short: "Update a Docker project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("description") {
				return fmt.Errorf("at least one of --name or --description is required")
			}
			projectID, err := projectIDFromArg(args)
			if err != nil {
				return err
			}
			req := api.UpdateDockerProjectRequest{}
			if cmd.Flags().Changed("name") {
				trimmed := strings.TrimSpace(name)
				if trimmed == "" {
					return fmt.Errorf("--name cannot be empty")
				}
				req.Name = &trimmed
			}
			if cmd.Flags().Changed("description") {
				req.Description = &description
			}

			client, serverID, err := dockerClient(opts.server)
			if err != nil {
				return err
			}
			project, err := client.UpdateDockerProject(serverID, projectID, req)
			if err != nil {
				return fmt.Errorf("failed to update Docker project: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, project)
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("Docker project %q updated", project.Name)))
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New project name")
	cmd.Flags().StringVar(&description, "description", "", "New description (empty clears it)")
	return cmd
}

func newProjectsDeleteCommand(opts *projectsOptions) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete [project-id]",
		Short: "Delete an empty Docker project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID, err := projectIDFromArg(args)
			if err != nil {
				return err
			}
			client, serverID, err := dockerClient(opts.server)
			if err != nil {
				return err
			}
			project, err := client.GetDockerProject(serverID, projectID)
			if err != nil {
				return fmt.Errorf("failed to get Docker project: %w", err)
			}
			if total := project.ApplicationsCount + project.ComposesCount + project.DatabasesCount; total > 0 {
				return fmt.Errorf("cannot delete Docker project %q: it still contains %d workload(s)", project.Name, total)
			}

			confirmed, err := confirmDestructive(
				cmd,
				yes,
				fmt.Sprintf("Delete Docker project %q?", project.Name),
				fmt.Sprintf("Project ID: %s. This action cannot be undone.", project.ID),
			)
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Fprintln(cmd.OutOrStdout(), tui.Dim.Render("Cancelled"))
				return nil
			}
			if err := client.DeleteDockerProject(serverID, project.ID); err != nil {
				return fmt.Errorf("failed to delete Docker project: %w", err)
			}
			if jsonEnabled(cmd) {
				return writeJSON(cmd, map[string]any{"deleted": true, "project_id": project.ID})
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.Success.Render(fmt.Sprintf("Docker project %q deleted", project.Name)))
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm deletion without prompting")
	return cmd
}
