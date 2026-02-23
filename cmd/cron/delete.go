package cron

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var deleteServerFlag string

var deleteCmd = &cobra.Command{
	Use:   "delete <cron-id>",
	Short: "Delete a cron job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(deleteServerFlag)
		if err != nil {
			return err
		}

		var confirm bool
		huh.NewConfirm().
			Title(fmt.Sprintf("Delete cron job %q?", args[0])).
			Description("This action cannot be undone.").
			Value(&confirm).
			Run()

		if !confirm {
			fmt.Println(tui.Dim.Render("Cancelled"))
			return nil
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		err = client.DeleteCron(serverID, args[0])
		if err != nil {
			return fmt.Errorf("failed to delete cron job: %w", err)
		}

		fmt.Println(tui.Success.Render("Cron job deleted"))
		return nil
	},
}

func init() {
	deleteCmd.Flags().StringVar(&deleteServerFlag, "server", "", "Server ID")
}
