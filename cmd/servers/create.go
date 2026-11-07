package servers

import (
	"fmt"
	"net"
	"strings"
	"unicode/utf8"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/commandutil"
	"github.com/spf13/cobra"
)

func newCreateOptionsCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "create-options",
		Short:        "Show supported server settings and provider plans as JSON",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			options, err := api.NewClient(appstate.GetConfig()).GetCreateServerOptionsContext(cmd.Context())
			if err != nil {
				return fmt.Errorf("get server creation options: %w", err)
			}
			return commandutil.WriteJSON(cmd, options)
		},
	}
}

func newCreateCommand() *cobra.Command {
	var req api.CreateServerRequest
	var description string
	var yes, installAgent bool
	cmd := &cobra.Command{
		Use:          "create",
		Short:        "Register a custom server or provision a cloud server",
		Long:         "Create a server using explicit settings. Cloud servers require a connected provider credential and may incur provider charges. Custom servers await setup and a successful try-connection. Use create-options for supported values.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			req.Name = strings.TrimSpace(req.Name)
			if err := validateCreateServer(req); err != nil {
				return err
			}
			if cmd.Flags().Changed("ssh-port") && (req.SSHPort < 1 || req.SSHPort > 65535) {
				return fmt.Errorf("--ssh-port must be between 1 and 65535")
			}
			if cmd.Flags().Changed("description") {
				req.Description = &description
			}
			if cmd.Flags().Changed("install-agent") {
				req.InstallAgent = &installAgent
			}
			confirmed, err := commandutil.Confirm(cmd, yes, "Create this server?", fmt.Sprintf("Create %q using %s (%s). Cloud providers may charge for the new server.", req.Name, req.Provider, req.Type))
			if err != nil {
				return err
			}
			if !confirmed {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
				return err
			}
			server, err := api.NewClient(appstate.GetConfig()).CreateServerContext(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}
			if commandutil.JSONEnabled(cmd) {
				return commandutil.WriteJSON(cmd, server)
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Server %q created (ID: %s, status: %s).\n", server.Name, server.ID, server.Status); err != nil {
				return err
			}
			if server.Status == "awaiting_connection" {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Complete the custom-server setup shown in the dashboard, then run lctl servers try-connection %s --yes.\n", server.ID)
			} else {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Provisioning is asynchronous; inspect progress with lctl servers show %s.\n", server.ID)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&req.Name, "name", "", "Server name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Server description")
	cmd.Flags().StringVar(&req.Provider, "provider", "", "custom_server, digitalocean, hetzner, linode, vultr, or aws (required)")
	cmd.Flags().StringVar(&req.Type, "type", "", "php, database, loadbalancer, or docker (required)")
	cmd.Flags().StringVar(&req.CredentialID, "credential-id", "", "Connected provider credential ID (required for cloud servers)")
	cmd.Flags().StringVar(&req.Region, "region", "", "Provider region (required for cloud servers)")
	cmd.Flags().StringVar(&req.Size, "size", "", "Provider plan (required for cloud servers)")
	cmd.Flags().StringVar(&req.IP, "ip", "", "Custom server IP address (required for custom_server)")
	cmd.Flags().IntVar(&req.SSHPort, "ssh-port", 0, "SSH port (server default: 22)")
	cmd.Flags().StringVar(&req.SSHUser, "ssh-user", "", "SSH username (uses server default when omitted)")
	cmd.Flags().StringVar(&req.OperatingSystem, "os", "", "Operating system key (server default: ubuntu_24)")
	cmd.Flags().StringVar(&req.PHPVersion, "php-version", "", "PHP software key, or none (uses server defaults when omitted)")
	cmd.Flags().StringVar(&req.DatabaseType, "database-type", "", "Database software key, or none (uses server defaults when omitted)")
	cmd.Flags().StringSliceVar(&req.SSHKeyIDs, "ssh-key", nil, "Saved SSH key IDs (repeatable or comma-separated)")
	cmd.Flags().BoolVar(&installAgent, "install-agent", true, "Install Launch Agent where supported by the server type")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm server creation without prompting")
	return cmd
}

func validateCreateServer(req api.CreateServerRequest) error {
	if length := utf8.RuneCountInString(req.Name); length < 2 || length > 255 {
		return fmt.Errorf("--name must contain between 2 and 255 characters")
	}
	switch req.Type {
	case "php", "database", "loadbalancer", "docker":
	default:
		return fmt.Errorf("--type must be php, database, loadbalancer, or docker")
	}
	switch req.Provider {
	case "custom_server":
		if net.ParseIP(req.IP) == nil {
			return fmt.Errorf("--ip must be a valid IP address for custom_server")
		}
		if req.CredentialID != "" || req.Region != "" || req.Size != "" {
			return fmt.Errorf("--credential-id, --region, and --size apply only to cloud servers")
		}
	case "digitalocean", "hetzner", "linode", "vultr", "aws":
		if strings.TrimSpace(req.CredentialID) == "" || strings.TrimSpace(req.Region) == "" || strings.TrimSpace(req.Size) == "" {
			return fmt.Errorf("--credential-id, --region, and --size are required for cloud servers")
		}
		if req.IP != "" {
			return fmt.Errorf("--ip applies only to custom_server")
		}
	default:
		return fmt.Errorf("--provider must be custom_server, digitalocean, hetzner, linode, vultr, or aws")
	}
	return nil
}
