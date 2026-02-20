package env

import (
	"fmt"
	"os"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var (
	pullServerFlag string
	pullSiteFlag   string
	pullOutputFlag string
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull .env file from server",
	Long:  "Download the .env file from a remote site. Prints to stdout by default, or writes to a file with -o.",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID, err := resolve.ServerID(pullServerFlag)
		if err != nil {
			return err
		}

		siteID, err := resolve.SiteID(pullSiteFlag)
		if err != nil {
			return err
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		envFile, err := findEnvFile(client, serverID, siteID)
		if err != nil {
			return err
		}

		content, err := client.GetFileContent(serverID, siteID, envFile.ShowRoute)
		if err != nil {
			return fmt.Errorf("failed to get .env content: %w", err)
		}

		if pullOutputFlag != "" {
			if err := os.WriteFile(pullOutputFlag, []byte(content.Content), 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			tui.ShowSuccess(fmt.Sprintf("Environment file written to %s", pullOutputFlag))
			return nil
		}

		fmt.Print(content.Content)
		return nil
	},
}

func findEnvFile(client *api.Client, serverID, siteID string) (*api.FileOnServer, error) {
	files, err := client.ListFiles(serverID, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	for _, f := range files {
		if f.Type == "environment" {
			return &f, nil
		}
	}

	return nil, fmt.Errorf("no .env file found for this site")
}

func init() {
	pullCmd.Flags().StringVar(&pullServerFlag, "server", "", "Server ID")
	pullCmd.Flags().StringVar(&pullSiteFlag, "site", "", "Site ID")
	pullCmd.Flags().StringVarP(&pullOutputFlag, "output", "o", "", "Write to file instead of stdout")
}
