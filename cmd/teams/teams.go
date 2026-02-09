package teams

import (
	"github.com/spf13/cobra"
)

var TeamsCmd = &cobra.Command{
	Use:     "teams",
	Aliases: []string{"team"},
	Short:   "Manage teams",
}

func init() {
	TeamsCmd.AddCommand(listCmd)
	TeamsCmd.AddCommand(switchCmd)
	TeamsCmd.AddCommand(membersCmd)
}
