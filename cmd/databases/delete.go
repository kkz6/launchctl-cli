package databases

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var deleteServerFlag string

var deleteCmd = &cobra.Command{
	Use:   "delete <database-id>",
	Short: "Delete a database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(deleteServerFlag)
		if err != nil {
			return err
		}

		var confirm bool
		huh.NewConfirm().
			Title(fmt.Sprintf("Delete database %q?", args[0])).
			Description("This action cannot be undone.").
			Value(&confirm).
			Run()

		if !confirm {
			fmt.Println(tui.Dim.Render("Cancelled"))
			return nil
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		err = client.DeleteDatabase(serverID, args[0])
		if err != nil {
			return fmt.Errorf("failed to delete database: %w", err)
		}

		fmt.Println(tui.Success.Render("Database deleted"))
		return nil
	},
}

func init() {
	deleteCmd.Flags().StringVar(&deleteServerFlag, "server", "", "Server ID")
}
