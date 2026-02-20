package daemons

import (
	"github.com/spf13/cobra"
)

var DaemonsCmd = &cobra.Command{
	Use:   "daemons",
	Short: "Manage daemons",
	Long:  "List, add, restart, and delete daemons on your servers.",
}

func init() {
	DaemonsCmd.AddCommand(listCmd)
	DaemonsCmd.AddCommand(addCmd)
	DaemonsCmd.AddCommand(restartCmd)
	DaemonsCmd.AddCommand(deleteCmd)
}
