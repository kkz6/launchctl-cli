package ssl

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/output"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

var (
	listServerFlag string
	listSiteFlag   string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List SSL certificates",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(listServerFlag)
		if err != nil {
			return err
		}

		siteID, err := resolve.SiteID(listSiteFlag)
		if err != nil {
			return err
		}

		cfg := appstate.GetConfig()
		client := api.NewClient(cfg)

		certs, err := client.ListCertificates(serverID, siteID)
		if err != nil {
			return fmt.Errorf("failed to list certificates: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, _ := json.MarshalIndent(certs, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		var rows [][]string
		for _, c := range certs {
			active := "No"
			if c.IsActive {
				active = "Yes"
			}

			domains := strings.Join(c.Domains, ", ")

			rows = append(rows, []string{
				c.ID,
				c.Type,
				domains,
				active,
				c.CreatedAt,
			})
		}

		output.RenderTable("SSL Certificates", []string{"ID", "Type", "Domains", "Active", "Created"}, rows)
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listServerFlag, "server", "", "Server ID")
	listCmd.Flags().StringVar(&listSiteFlag, "site", "", "Site ID")
}
