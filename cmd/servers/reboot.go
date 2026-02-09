package servers

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var rebootCmd = &cobra.Command{
	Use:   "reboot <id>",
	Short: "Reboot a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		server, err := client.GetServer(args[0])
		if err != nil {
			return fmt.Errorf("failed to get server: %w", err)
		}

		var confirm bool
		huh.NewConfirm().
			Title(fmt.Sprintf("Reboot server %q?", server.Name)).
			Description("This will restart the server and cause brief downtime.").
			Value(&confirm).
			Run()

		if !confirm {
			fmt.Println(tui.Dim.Render("Cancelled"))
			return nil
		}

		if err := client.RebootServer(args[0]); err != nil {
			return fmt.Errorf("failed to reboot server: %w", err)
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Server %q is rebooting", server.Name)))
		return nil
	},
}
