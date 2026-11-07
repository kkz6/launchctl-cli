package servers

import (
	"github.com/kkz6/launchctl/internal/api"
	"github.com/spf13/cobra"
)

var rebootCmd = newRebootCommand()

func newRebootCommand() *cobra.Command {
	return newServerActionCommand(serverAction{
		name:        "reboot",
		description: "Reboot the server, causing brief downtime",
		status:      "queued",
		confirm:     true,
		run:         (*api.Client).RebootServerContext,
	})
}
