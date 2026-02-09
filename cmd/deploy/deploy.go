package deploy

import (
	"github.com/spf13/cobra"
)

var DeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy sites and manage deployments",
}

func init() {
	DeployCmd.AddCommand(triggerCmd)
	DeployCmd.AddCommand(listCmd)
	DeployCmd.AddCommand(showCmd)
	DeployCmd.AddCommand(rollbackCmd)
}
