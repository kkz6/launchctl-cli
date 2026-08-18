package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var DockerCmd = newDockerCommand()

func newDockerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Manage Docker projects and applications",
		Long:  "Manage projects and single-container applications on Docker application servers.",
	}
	cmd.AddCommand(newProjectsCommand())
	cmd.AddCommand(newApplicationsCommand())
	return cmd
}

func dockerClient(serverFlag string) (*api.Client, string, error) {
	serverID, err := resolve.ServerID(strings.TrimSpace(serverFlag))
	if err != nil {
		return nil, "", err
	}

	client := api.NewClient(appstate.GetConfig())
	server, err := client.GetServer(serverID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get server: %w", err)
	}
	if server.Type != "docker" {
		return nil, "", fmt.Errorf("server %q is not a Docker application server", server.Name)
	}
	return client, serverID, nil
}

func projectIDFromArg(args []string) (string, error) {
	value := ""
	if len(args) > 0 {
		value = strings.TrimSpace(args[0])
	}
	return resolve.ProjectID(value)
}

func applicationIDFromArg(args []string) (string, error) {
	value := ""
	if len(args) > 0 {
		value = strings.TrimSpace(args[0])
	}
	return resolve.ApplicationID(value)
}

func jsonEnabled(cmd *cobra.Command) bool {
	value, _ := cmd.Flags().GetBool("json")
	return value
}

func ciEnabled(cmd *cobra.Command) bool {
	value, _ := cmd.Flags().GetBool("ci")
	return value
}

func writeJSON(cmd *cobra.Command, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JSON output: %w", err)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}

func confirmDestructive(cmd *cobra.Command, yes bool, title, description string) (bool, error) {
	if yes {
		return true, nil
	}
	if ciEnabled(cmd) || jsonEnabled(cmd) || !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return false, fmt.Errorf("confirmation required; rerun with --yes")
	}

	confirmed := false
	err := huh.NewConfirm().
		Title(title).
		Description(description).
		Value(&confirmed).
		WithTheme(tui.FormTheme()).
		Run()
	if err != nil {
		return false, err
	}
	return confirmed, nil
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Local().Format("2006-01-02 15:04:05")
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
