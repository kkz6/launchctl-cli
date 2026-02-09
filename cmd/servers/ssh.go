package servers

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/spf13/cobra"
)

var sshUser string

var sshCmd = &cobra.Command{
	Use:   "ssh <id>",
	Short: "SSH into a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		server, err := client.GetServer(args[0])
		if err != nil {
			return fmt.Errorf("failed to get server: %w", err)
		}

		if server.PublicIPv4 == nil {
			return fmt.Errorf("server has no public IP address")
		}

		user := sshUser
		if user == "" {
			user = server.Username
		}

		sshArgs := []string{
			"-p", strconv.Itoa(server.SSHPort),
			fmt.Sprintf("%s@%s", user, *server.PublicIPv4),
		}

		sshBin, err := exec.LookPath("ssh")
		if err != nil {
			return fmt.Errorf("ssh not found in PATH")
		}

		c := exec.Command(sshBin, sshArgs...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		return c.Run()
	},
}

func init() {
	sshCmd.Flags().StringVarP(&sshUser, "user", "u", "", "SSH user (default: server username)")
}
