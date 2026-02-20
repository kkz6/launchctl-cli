package cron

import (
	"github.com/spf13/cobra"
)

var CronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage cron jobs",
	Long:  "List, add, and delete cron jobs on your servers.",
}

func init() {
	CronCmd.AddCommand(listCmd)
	CronCmd.AddCommand(addCmd)
	CronCmd.AddCommand(deleteCmd)
}
