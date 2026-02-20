package databases

import (
	"github.com/spf13/cobra"
)

var DatabasesCmd = &cobra.Command{
	Use:   "databases",
	Short: "Manage server databases",
	Long:  "List, create, and delete databases on your servers.",
}

func init() {
	DatabasesCmd.AddCommand(listCmd)
	DatabasesCmd.AddCommand(createCmd)
	DatabasesCmd.AddCommand(deleteCmd)
	DatabasesCmd.AddCommand(usersCmd)
}
