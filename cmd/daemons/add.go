package daemons

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var addServerFlag string

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(addServerFlag)
		if err != nil {
			return err
		}

		var (
			command      string
			user         string
			directory    string
			processesStr string
		)

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Command").
					Value(&command),
				huh.NewInput().
					Title("User").
					Description("e.g. root, forge").
					Value(&user),
				huh.NewInput().
					Title("Directory (optional)").
					Value(&directory),
				huh.NewInput().
					Title("Processes").
					Description("Number of processes (default: 1)").
					Value(&processesStr),
			),
		).Run()
		if err != nil {
			return nil
		}

		processes := 1
		if processesStr != "" {
			processes, err = strconv.Atoi(processesStr)
			if err != nil {
				return fmt.Errorf("invalid number of processes: %w", err)
			}
		}

		req := api.CreateDaemonRequest{
			Command:   command,
			User:      user,
			Processes: processes,
		}
		if directory != "" {
			req.Directory = &directory
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		daemon, err := client.CreateDaemon(serverID, req)
		if err != nil {
			return fmt.Errorf("failed to add daemon: %w", err)
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Daemon added (ID: %s)", daemon.ID)))
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addServerFlag, "server", "", "Server ID")
}
