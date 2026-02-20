package sshkeys

import (
	"github.com/spf13/cobra"
)

var SSHKeysCmd = &cobra.Command{
	Use:   "ssh-keys",
	Short: "Manage SSH keys",
	Long:  "List, add, and manage SSH keys for your team and servers.",
}

func init() {
	SSHKeysCmd.AddCommand(listCmd)
	SSHKeysCmd.AddCommand(addCmd)
	SSHKeysCmd.AddCommand(deleteCmd)
	SSHKeysCmd.AddCommand(serverListCmd)
	SSHKeysCmd.AddCommand(attachCmd)
	SSHKeysCmd.AddCommand(detachCmd)
}
