package firewall

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var addServerFlag string

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a firewall rule",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(addServerFlag)
		if err != nil {
			return err
		}

		var (
			name   string
			action string
			port   string
			fromIP string
			note   string
		)

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Rule Name").
					Value(&name),
				huh.NewSelect[string]().
					Title("Action").
					Options(
						huh.NewOption("Allow", "allow"),
						huh.NewOption("Deny", "deny"),
					).
					Value(&action),
				huh.NewInput().
					Title("Port").
					Description("e.g. 80, 443, 8080").
					Value(&port),
				huh.NewInput().
					Title("From IP (optional)").
					Value(&fromIP),
				huh.NewInput().
					Title("Note (optional)").
					Value(&note),
			),
		).Run()
		if err != nil {
			return nil
		}

		req := api.CreateFirewallRuleRequest{
			Name:   name,
			Action: action,
			Port:   port,
		}
		if fromIP != "" {
			req.FromIPv4 = &fromIP
		}
		if note != "" {
			req.Note = &note
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		rule, err := client.CreateFirewallRule(serverID, req)
		if err != nil {
			return fmt.Errorf("failed to add firewall rule: %w", err)
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("Firewall rule %q added (ID: %s)", rule.Name, rule.ID)))
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addServerFlag, "server", "", "Server ID")
}
