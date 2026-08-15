package tasks

import "github.com/spf13/cobra"

var serverFlag string

var TasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Inspect and watch server tasks",
}

func init() {
	TasksCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "Server ID (or project default)")
	TasksCmd.AddCommand(listCmd)
	TasksCmd.AddCommand(watchCmd)
}
