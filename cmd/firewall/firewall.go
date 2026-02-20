package firewall

import (
	"github.com/spf13/cobra"
)

var FirewallCmd = &cobra.Command{
	Use:   "firewall",
	Short: "Manage firewall rules",
	Long:  "List, add, and delete firewall rules on your servers.",
}

func init() {
	FirewallCmd.AddCommand(listCmd)
	FirewallCmd.AddCommand(addCmd)
	FirewallCmd.AddCommand(deleteCmd)
}
