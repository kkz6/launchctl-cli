package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of launchctl",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("launchctl %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
