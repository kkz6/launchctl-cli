package sshkeys

import (
	"fmt"
	"os"
	"strings"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/kkz6/launchctl/internal/tui"
	"github.com/spf13/cobra"
)

var (
	addNameFlag   string
	addKeyFlag    string
	addGlobalFlag bool
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an SSH key",
	Long:  "Add an SSH key by specifying a name and the path to a public key file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if addNameFlag == "" {
			return fmt.Errorf("--name is required")
		}
		if addKeyFlag == "" {
			return fmt.Errorf("--key is required (path to public key file)")
		}

		data, err := os.ReadFile(addKeyFlag)
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		publicKey := strings.TrimSpace(string(data))

		cfg, _ := config.Load()
		client := api.NewClient(cfg)

		key, err := client.CreateSSHKey(api.CreateSSHKeyRequest{
			Name:      addNameFlag,
			PublicKey: publicKey,
			IsGlobal:  addGlobalFlag,
		})
		if err != nil {
			return fmt.Errorf("failed to add SSH key: %w", err)
		}

		fmt.Println(tui.Success.Render(fmt.Sprintf("SSH key %q added (ID: %s)", key.Name, key.ID)))
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addNameFlag, "name", "", "Name for the SSH key")
	addCmd.Flags().StringVar(&addKeyFlag, "key", "", "Path to the public key file")
	addCmd.Flags().BoolVar(&addGlobalFlag, "global", false, "Make the key available to all servers")
}
