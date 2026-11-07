package sites

import (
	"fmt"
	"strings"

	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

var SitesCmd = &cobra.Command{
	Use:     "sites",
	Aliases: []string{"site"},
	Short:   "Manage sites",
}

func init() {
	SitesCmd.AddCommand(listCmd)
	SitesCmd.AddCommand(showCmd)
	SitesCmd.AddCommand(newCreateCommand())
	SitesCmd.AddCommand(newUpdateCommand())
	SitesCmd.AddCommand(newDeleteCommand())
	SitesCmd.AddCommand(newAutoDeployCommand())
}

func siteTarget(serverFlag string, args []string) (string, string, error) {
	serverID, err := resolve.ServerID(strings.TrimSpace(serverFlag))
	if err != nil {
		return "", "", err
	}
	value := ""
	if len(args) > 0 {
		value = strings.TrimSpace(args[0])
	}
	siteID, err := resolve.SiteID(value)
	if err != nil {
		return "", "", fmt.Errorf("site ID is required (or run `lctl init` to set a default)")
	}
	return serverID, siteID, nil
}
