package ssl

import (
	"github.com/spf13/cobra"
)

var SSLCmd = &cobra.Command{
	Use:   "ssl",
	Short: "Manage SSL certificates",
	Long:  "List SSL certificates for your sites.",
}

func init() {
	SSLCmd.AddCommand(listCmd)
}
