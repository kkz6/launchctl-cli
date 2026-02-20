package firewall

import (
	"encoding/json"
	"fmt"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

var listServerFlag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List firewall rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(listServerFlag)
		if err != nil {
			return err
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		rules, err := client.ListFirewallRules(serverID)
		if err != nil {
			return fmt.Errorf("failed to list firewall rules: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(rules, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		var rows [][]string
		for _, r := range rules {
			port := ""
			if r.Port != nil {
				port = *r.Port
			}
			fromIP := ""
			if r.FromIPv4 != nil {
				fromIP = *r.FromIPv4
			}

			status := "installed"
			if r.HasFailed {
				status = "failed"
			} else if r.IsPending {
				status = "pending"
			} else if !r.IsInstalled {
				status = "not installed"
			}

			rows = append(rows, []string{
				r.Name,
				r.ActionLabel,
				port,
				fromIP,
				output.StatusDot(status),
			})
		}

		output.RenderTable("Firewall Rules", []string{"Name", "Action", "Port", "From IP", "Status"}, rows)
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listServerFlag, "server", "", "Server ID")
}
