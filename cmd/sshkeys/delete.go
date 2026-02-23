package sshkeys

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <key-id>",
	Short: "Delete an SSH key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var confirm bool
		huh.NewConfirm().
			Title(fmt.Sprintf("Delete SSH key %q?", args[0])).
			Description("This action cannot be undone.").
			Value(&confirm).
			Run()

		if !confirm {
			fmt.Println(tui.Dim.Render("Cancelled"))
			return nil
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		err := client.DeleteSSHKey(args[0])
		if err != nil {
			return fmt.Errorf("failed to delete SSH key: %w", err)
		}

		fmt.Println(tui.Success.Render("SSH key deleted"))
		return nil
	},
}
