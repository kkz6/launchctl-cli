package servers

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/terminal"
	"github.com/spf13/cobra"
)

var sshUser string

var sshCmd = &cobra.Command{
	Use:   "ssh <id>",
	Short: "Open a terminal session on a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		server, err := client.GetServer(args[0])
		if err != nil {
			return fmt.Errorf("failed to get server: %w", err)
		}

		if !server.Connected {
			return fmt.Errorf("server %q is not connected", server.Name)
		}

		user := sshUser
		if user == "" {
			user = server.Username
		}

		jwt, err := client.ExchangeToken()
		if err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}

		fmt.Printf("Connecting to %s...\n", server.Name)

		return terminal.Connect(cfg, terminal.Options{
			ServerID: server.ID,
			Username: user,
			Token:    jwt,
		})
	},
}

func init() {
	sshCmd.Flags().StringVarP(&sshUser, "user", "u", "", "SSH user (default: server username)")
}
