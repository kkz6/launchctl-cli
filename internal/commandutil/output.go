package commandutil

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

func JSONEnabled(cmd *cobra.Command) bool {
	enabled, _ := cmd.Flags().GetBool("json")
	return enabled
}

func WriteJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// Confirm keeps scripts from silently skipping a requested destructive action.
func Confirm(cmd *cobra.Command, yes bool, title, description string) (bool, error) {
	if yes {
		return true, nil
	}
	ci, _ := cmd.Flags().GetBool("ci")
	if ci || JSONEnabled(cmd) || !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return false, fmt.Errorf("confirmation required; rerun with --yes")
	}
	confirmed := false
	err := huh.NewConfirm().Title(title).Description(description).
		Value(&confirmed).WithTheme(tui.FormTheme()).Run()
	return confirmed, err
}
