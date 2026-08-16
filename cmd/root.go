package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/kkz6/launchctl/cmd/cron"
	"github.com/kkz6/launchctl/cmd/daemons"
	"github.com/kkz6/launchctl/cmd/databases"
	"github.com/kkz6/launchctl/cmd/deploy"
	"github.com/kkz6/launchctl/cmd/env"
	"github.com/kkz6/launchctl/cmd/firewall"
	"github.com/kkz6/launchctl/cmd/servers"
	"github.com/kkz6/launchctl/cmd/services"
	"github.com/kkz6/launchctl/cmd/sites"
	"github.com/kkz6/launchctl/cmd/sshkeys"
	"github.com/kkz6/launchctl/cmd/ssl"
	"github.com/kkz6/launchctl/cmd/tasks"
	"github.com/kkz6/launchctl/cmd/teams"
	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/appstate"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/splash"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/kkz6/launchctl/internal/tui/nav"
	"github.com/spf13/cobra"
)

var (
	Version = "0.1.0"

	jsonOutput  bool
	ciMode      bool
	profileFlag string
	apiURLFlag  string
	cfg         *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "lctl",
	Short: "CLI for managing launchctl servers, sites, and deployments",
	Long:  "lctl is a command-line tool for managing your launchctl servers, sites, and deployments.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if commandSkipsConfig(cmd) {
			cfg = config.DefaultConfig()
			appstate.SetConfig(cfg)
			return nil
		}

		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if profileFlag != "" {
			if err := cfg.ActivateProfile(profileFlag); err != nil {
				return fmt.Errorf("--profile: %w", err)
			}
		}

		cfg.ApplyEnvOverrides()
		if apiURLFlag != "" {
			cfg.APIURL = apiURLFlag
		}

		appstate.SetConfig(cfg)

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		tui.ClearScreen()
		fmt.Print(splash.Render(Version))
		fmt.Println()
		time.Sleep(2 * time.Second)
		tui.ClearScreen()

		if cfg.IsAuthenticated() {
			client := api.NewClient(cfg)
			nav.Run(client, cfg)
		} else {
			cmd.Help()
		}
	},
}

func commandSkipsConfig(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		if current.Annotations["skipConfig"] == "true" {
			return true
		}
	}
	return false
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&ciMode, "ci", false, "CI/CD mode (non-interactive, requires flags/env vars)")
	rootCmd.PersistentFlags().StringVar(&profileFlag, "profile", "", "Use a specific profile for this command")
	rootCmd.PersistentFlags().StringVar(&apiURLFlag, "api-url", "", "Override the launchctl API URL for development or testing")

	rootCmd.AddCommand(servers.ServersCmd)
	rootCmd.AddCommand(sites.SitesCmd)
	rootCmd.AddCommand(deploy.DeployCmd)
	rootCmd.AddCommand(teams.TeamsCmd)
	rootCmd.AddCommand(tasks.TasksCmd)
	rootCmd.AddCommand(env.EnvCmd)
	rootCmd.AddCommand(services.ServicesCmd)
	rootCmd.AddCommand(databases.DatabasesCmd)
	rootCmd.AddCommand(sshkeys.SSHKeysCmd)
	rootCmd.AddCommand(firewall.FirewallCmd)
	rootCmd.AddCommand(cron.CronCmd)
	rootCmd.AddCommand(ssl.SSLCmd)
	rootCmd.AddCommand(daemons.DaemonsCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(runCmd)
}

func GetConfig() *config.Config {
	return cfg
}

func IsJSON() bool {
	return jsonOutput
}

func IsCI() bool {
	return ciMode
}
