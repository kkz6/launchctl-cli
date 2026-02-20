package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of lctl",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("lctl %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
