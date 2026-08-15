package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	launchapi "github.com/kkz6/launchctl/internal/api"
	"github.com/spf13/cobra"
)

var apiData string

var apiCmd = &cobra.Command{
	Use:   "api <method> <path>",
	Short: "Call an authenticated launchctl API endpoint",
	Long: `Call any launchctl API endpoint with the active profile's credentials.

This is the forward-compatible escape hatch for endpoints that do not yet
have a dedicated command. The path should include /api. Pass a JSON object or
array with --data, or use --data @file.json.`,
	Example: `  lctl api GET /api/servers
  lctl api GET '/api/docker/projects?server_id=01ABC'
  lctl api POST /api/scripts --data '{"name":"health-check"}'
  lctl api PATCH /api/notifications/settings --data @settings.json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cfg.IsAuthenticated() {
			return fmt.Errorf("not authenticated — run `lctl login` first")
		}

		method := strings.ToUpper(strings.TrimSpace(args[0]))
		path := strings.TrimSpace(args[1])
		if !strings.HasPrefix(path, "/api/") && path != "/api" {
			return fmt.Errorf("API path must start with /api")
		}

		body, err := parseAPIData(apiData)
		if err != nil {
			return err
		}

		client := launchapi.NewClient(cfg)
		data, _, err := client.RawRequest(cmd.Context(), method, path, body)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(data)) == 0 {
			return nil
		}

		var value any
		if json.Unmarshal(data, &value) == nil {
			formatted, err := json.MarshalIndent(value, "", "  ")
			if err == nil {
				fmt.Println(string(formatted))
				return nil
			}
		}
		fmt.Print(string(data))
		if data[len(data)-1] != '\n' {
			fmt.Println()
		}
		return nil
	},
}

func parseAPIData(input string) (any, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}
	data := []byte(input)
	if strings.HasPrefix(input, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(input, "@"))
		if path == "" {
			return nil, fmt.Errorf("--data @file requires a file path")
		}
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read API request body: %w", err)
		}
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("--data must contain valid JSON: %w", err)
	}
	return value, nil
}

func init() {
	apiCmd.Flags().StringVarP(&apiData, "data", "d", "", "JSON request body or @file.json")
	rootCmd.AddCommand(apiCmd)
}
