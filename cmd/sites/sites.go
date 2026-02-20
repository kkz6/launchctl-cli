package sites

import (
	"github.com/spf13/cobra"
)

var SitesCmd = &cobra.Command{
	Use:     "sites",
	Aliases: []string{"site"},
	Short:   "Manage sites",
}

func init() {
	SitesCmd.AddCommand(listCmd)
	SitesCmd.AddCommand(showCmd)
}
