package deploy

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var DeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy sites and manage deployments",
}

func init() {
	DeployCmd.AddCommand(newTriggerCommand())
	DeployCmd.AddCommand(listCmd)
	DeployCmd.AddCommand(showCmd)
	DeployCmd.AddCommand(newRollbackCommand())
	DeployCmd.AddCommand(newCancelQueuedCommand())
	DeployCmd.AddCommand(logsCmd)
}

func jsonEnabled(cmd *cobra.Command) bool {
	value, _ := cmd.Flags().GetBool("json")
	return value
}

func ciEnabled(cmd *cobra.Command) bool {
	value, _ := cmd.Flags().GetBool("ci")
	return value
}

func interactive(cmd *cobra.Command) bool {
	return !jsonEnabled(cmd) && !ciEnabled(cmd) && term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
}

func writeJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func confirmDestructive(cmd *cobra.Command, yes bool, title, description string) (bool, error) {
	if yes {
		return true, nil
	}
	if !interactive(cmd) {
		return false, fmt.Errorf("confirmation required; rerun with --yes")
	}
	confirmed := false
	err := huh.NewConfirm().Title(title).Description(description).
		Value(&confirmed).WithTheme(tui.FormTheme()).Run()
	return confirmed, err
}
