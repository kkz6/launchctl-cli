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

var addServerFlag string

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a cron job",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(addServerFlag)
		if err != nil {
			return err
		}

		var (
			user       string
			expression string
			command    string
		)

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("User").
					Description("e.g. root, forge").
					Value(&user),
				huh.NewInput().
					Title("Cron Expression").
					Description("e.g. * * * * *").
					Value(&expression),
				huh.NewInput().
					Title("Command").
					Value(&command),
			),
		).Run()
		if err != nil {
			return nil
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		req := api.CreateCronRequest{
			User:       user,
			Expression: expression,
			Command:    command,
		}

		cronJob, err := client.CreateCron(serverID, req)
		if err != nil {
			return fmt.Errorf("failed to add cron job: %w", err)
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Cron job added (ID: %s)", cronJob.ID)))
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addServerFlag, "server", "", "Server ID")
}
