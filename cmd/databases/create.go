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

var createServerFlag string

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new database",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(createServerFlag)
		if err != nil {
			return err
		}

		var (
			dbName     string
			createUser bool
			userName   string
			password   string
		)

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Database Name").
					Value(&dbName),
				huh.NewConfirm().
					Title("Create a database user?").
					Value(&createUser),
			),
		).Run()
		if err != nil {
			return nil
		}

		if createUser {
			err = huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Username").
						Value(&userName),
					huh.NewInput().
						Title("Password").
						EchoMode(huh.EchoModePassword).
						Value(&password),
				),
			).Run()
			if err != nil {
				return nil
			}
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		req := api.CreateDatabaseRequest{
			Name:         dbName,
			CreateUser:   createUser,
			UserName:     userName,
			UserPassword: password,
		}

		db, err := client.CreateDatabase(serverID, req)
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Database %q created (ID: %s)", db.Name, db.ID)))
		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&createServerFlag, "server", "", "Server ID")
}
