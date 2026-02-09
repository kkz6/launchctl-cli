package cmd

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and clear credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cfg.IsAuthenticated() {
			fmt.Println(tui.Dim.Render("Not logged in"))
			return nil
		}

		client := api.NewClient(cfg)
		_ = client.Logout()

		cfg.ClearAuth()
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to clear credentials: %w", err)
		}

		fmt.Println(tui.Success.Render("Logged out successfully"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
