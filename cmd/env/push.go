package env

import (
	"fmt"
	"os"
	"strings"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/resolve"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var (
	pushServerFlag string
	pushSiteFlag   string
	pushFileFlag   string
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push .env file to server",
	Long:  "Upload a local .env file to a remote site. Shows a diff and asks for confirmation before pushing.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if pushFileFlag == "" {
			return fmt.Errorf("--file flag is required")
		}

		serverID, err := resolve.ServerID(pushServerFlag)
		if err != nil {
			return err
		}

		siteID, err := resolve.SiteID(pushSiteFlag)
		if err != nil {
			return err
		}

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		localContent, err := os.ReadFile(pushFileFlag)
		if err != nil {
			return fmt.Errorf("failed to read local file: %w", err)
		}

		envFile, err := findEnvFile(client, serverID, siteID)
		if err != nil {
			return err
		}

		remote, err := client.GetFileContent(serverID, siteID, envFile.ShowRoute)
		if err != nil {
			return fmt.Errorf("failed to get remote .env: %w", err)
		}

		localStr := string(localContent)

		if strings.TrimSpace(localStr) == strings.TrimSpace(remote.Content) {
			tui.ShowInfo("No changes detected — remote .env is identical to local file")
			return nil
		}

		ciMode, _ := cmd.Flags().GetBool("ci")

		if !ciMode {
			showDiff(remote.Content, localStr)

			fmt.Print("\nPush these changes? [y/N] ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println(tui.Dim.Render("Aborted."))
				return nil
			}
		}

		err = client.UpdateFileContent(serverID, siteID, envFile.UpdateRoute, api.UpdateFileRequest{
			Content: localStr,
		})
		if err != nil {
			return fmt.Errorf("failed to push .env: %w", err)
		}

		tui.ShowSuccess("Environment file updated successfully")
		return nil
	},
}

func showDiff(remote, local string) {
	remoteLines := strings.Split(strings.TrimSpace(remote), "\n")
	localLines := strings.Split(strings.TrimSpace(local), "\n")

	remoteSet := make(map[string]string)
	for _, line := range remoteLines {
		if key, _, ok := parseEnvLine(line); ok {
			remoteSet[key] = line
		}
	}

	localSet := make(map[string]string)
	for _, line := range localLines {
		if key, _, ok := parseEnvLine(line); ok {
			localSet[key] = line
		}
	}

	fmt.Println()
	fmt.Println(tui.Title.Render("Changes"))
	fmt.Println()

	for _, line := range localLines {
		key, _, ok := parseEnvLine(line)
		if !ok {
			continue
		}
		if remoteLine, exists := remoteSet[key]; exists {
			if remoteLine != line {
				fmt.Println(tui.Warning.Render("~ " + line))
			}
		} else {
			fmt.Println(tui.Success.Render("+ " + line))
		}
	}

	for _, line := range remoteLines {
		key, _, ok := parseEnvLine(line)
		if !ok {
			continue
		}
		if _, exists := localSet[key]; !exists {
			fmt.Println(tui.Error.Render("- " + line))
		}
	}
}

func parseEnvLine(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}

	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

func init() {
	pushCmd.Flags().StringVar(&pushServerFlag, "server", "", "Server ID")
	pushCmd.Flags().StringVar(&pushSiteFlag, "site", "", "Site ID")
	pushCmd.Flags().StringVarP(&pushFileFlag, "file", "f", "", "Local .env file to push (required)")
}
