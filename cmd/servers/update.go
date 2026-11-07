package servers

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/commandutil"
	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	var name, description string
	var monitoring, autoUpdate bool
	var sshPort int
	cmd := &cobra.Command{
		Use:          "update [server-id]",
		Short:        "Update settings for a provisioned server",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := api.UpdateServerRequest{}
			if cmd.Flags().Changed("name") {
				name = strings.TrimSpace(name)
				if length := utf8.RuneCountInString(name); length < 2 || length > 255 {
					return fmt.Errorf("--name must contain between 2 and 255 characters")
				}
				req.Name = &name
			}
			if cmd.Flags().Changed("description") {
				req.Description = &description
			}
			if cmd.Flags().Changed("monitoring") {
				req.MonitoringEnabled = &monitoring
			}
			if cmd.Flags().Changed("auto-update") {
				req.AutoUpdate = &autoUpdate
			}
			if cmd.Flags().Changed("ssh-port") {
				if sshPort < 1 || sshPort > 65535 {
					return fmt.Errorf("--ssh-port must be between 1 and 65535")
				}
				req.SSHPort = &sshPort
			}
			if req.Name == nil && req.Description == nil && req.MonitoringEnabled == nil && req.AutoUpdate == nil && req.SSHPort == nil {
				return fmt.Errorf("provide at least one setting: --name, --description, --monitoring, --auto-update, or --ssh-port")
			}
			serverID, err := serverIDFromArgs(args)
			if err != nil {
				return err
			}
			server, err := api.NewClient(appstate.GetConfig()).UpdateServer(cmd.Context(), serverID, req)
			if err != nil {
				return fmt.Errorf("update server: %w", err)
			}
			if commandutil.JSONEnabled(cmd) {
				return commandutil.WriteJSON(cmd, server)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Server %q settings updated (ID: %s).\n", server.Name, server.ID)
			return err
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New server name")
	cmd.Flags().StringVar(&description, "description", "", "New description (empty clears it)")
	cmd.Flags().BoolVar(&monitoring, "monitoring", false, "Enable monitoring (use --monitoring=false to disable)")
	cmd.Flags().BoolVar(&autoUpdate, "auto-update", false, "Enable automatic updates (use --auto-update=false to disable)")
	cmd.Flags().IntVar(&sshPort, "ssh-port", 0, "Recorded SSH port; does not reconfigure SSH on the server")
	return cmd
}
