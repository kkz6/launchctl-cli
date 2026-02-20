package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/tui/dashboard"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show dashboard overview",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cfg.IsAuthenticated() {
			return fmt.Errorf("not logged in, run: lctl login")
		}

		client := api.NewClient(cfg)
		model := dashboard.NewModel(client, cfg.TeamName)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
