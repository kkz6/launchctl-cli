package services

import (
	"github.com/spf13/cobra"
)

var ServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Manage server services",
	Long:  "List and manage services installed on your servers.",
}

func init() {
	ServicesCmd.AddCommand(listCmd)
	ServicesCmd.AddCommand(restartCmd)
	ServicesCmd.AddCommand(stopCmd)
	ServicesCmd.AddCommand(startCmd)
}
