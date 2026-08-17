package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:         "version",
	Short:       "Print the version of lctl",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"version": Version})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "lctl %s\n", Version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
