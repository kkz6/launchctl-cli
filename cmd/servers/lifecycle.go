package servers

import (
	"context"
	"fmt"
	"strings"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/commandutil"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/spf13/cobra"
)

type serverAction struct {
	name        string
	description string
	status      string
	confirm     bool
	run         func(*api.Client, context.Context, string) error
}

var lifecycleActions = []serverAction{
	{"retry-provision", "Retry provisioning for a failed server", "queued", true, (*api.Client).RetryServerProvision},
	{"try-connection", "Check a custom server's SSH connection and start provisioning", "queued", true, (*api.Client).TryServerConnection},
	{"archive", "Archive the server record without stopping the server", "completed", true, (*api.Client).ArchiveServer},
	{"unarchive", "Restore an archived server record", "completed", false, (*api.Client).UnarchiveServer},
}

type serverActionReceipt struct {
	ServerID string `json:"server_id"`
	Action   string `json:"action"`
	Status   string `json:"status"`
}

func serverIDFromArgs(args []string) (string, error) {
	value := ""
	if len(args) > 0 {
		value = strings.TrimSpace(args[0])
		if value == "" {
			return "", fmt.Errorf("server ID cannot be empty")
		}
	}
	return resolve.ServerID(value)
}

func newServerActionCommand(action serverAction) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:          action.name + " [server-id]",
		Short:        action.description,
		Long:         action.description + ". The server defaults to .launchctl.yml when omitted.",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID, err := serverIDFromArgs(args)
			if err != nil {
				return err
			}
			if action.confirm {
				confirmed, err := commandutil.Confirm(cmd, yes, action.description+"?", fmt.Sprintf("Server: %s", serverID))
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
					return err
				}
			}
			if err := action.run(api.NewClient(appstate.GetConfig()), cmd.Context(), serverID); err != nil {
				return fmt.Errorf("%s server: %w", action.name, err)
			}
			receipt := serverActionReceipt{ServerID: serverID, Action: action.name, Status: action.status}
			if commandutil.JSONEnabled(cmd) {
				return commandutil.WriteJSON(cmd, receipt)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Server %s: %s %s.\n", serverID, action.name, action.status)
			return err
		},
	}
	if action.confirm {
		cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm the action without prompting")
	}
	return cmd
}
