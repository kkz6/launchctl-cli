package env

import (
	"github.com/spf13/cobra"
)

var EnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment files",
	Long:  "Push and pull .env files between your local machine and remote servers.",
}

func init() {
	EnvCmd.AddCommand(pullCmd)
	EnvCmd.AddCommand(pushCmd)
}
