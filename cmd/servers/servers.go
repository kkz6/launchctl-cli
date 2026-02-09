package servers

import (
	"github.com/spf13/cobra"
)

var ServersCmd = &cobra.Command{
	Use:     "servers",
	Aliases: []string{"server"},
	Short:   "Manage servers",
}

func init() {
	ServersCmd.AddCommand(listCmd)
	ServersCmd.AddCommand(showCmd)
	ServersCmd.AddCommand(createCmd)
	ServersCmd.AddCommand(rebootCmd)
	ServersCmd.AddCommand(sshCmd)
	ServersCmd.AddCommand(metricsCmd)
}
