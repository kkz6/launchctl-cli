package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/kkz6/launchctl/internal/aiskill"
	"github.com/spf13/cobra"
)

var (
	aiCodexHome      string
	aiUpdateForce    bool
	aiUninstallForce bool
)

type aiCommandResult struct {
	Action    string `json:"action"`
	Changed   bool   `json:"changed"`
	CodexCLI  bool   `json:"codex_cli_available,omitempty"`
	CodexPath string `json:"codex_cli_path,omitempty"`
	aiskill.Report
}

var aiCmd = &cobra.Command{
	Use:         "ai",
	Short:       "Manage the operate-launchctl AI skill",
	Long:        "Install and maintain the bundled operate-launchctl skill for Codex.",
	Annotations: map[string]string{"skipConfig": "true"},
}

var aiInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the bundled AI skill for Codex",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := newAISkillManager()
		if err != nil {
			return err
		}
		report, changed, err := manager.Install()
		if err != nil {
			return err
		}
		return writeAIResult(cmd, aiCommandResult{
			Action:  "install",
			Changed: changed,
			Report:  report,
		})
	},
}

var aiDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the bundled AI skill installation",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := newAISkillManager()
		if err != nil {
			return err
		}
		report, err := manager.Inspect()
		if err != nil {
			return err
		}
		codexPath, lookupErr := exec.LookPath("codex")
		return writeAIResult(cmd, aiCommandResult{
			Action:    "doctor",
			CodexCLI:  lookupErr == nil,
			CodexPath: codexPath,
			Report:    report,
		})
	},
}

var aiUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an lctl-managed AI skill installation",
	Long:  "Update the installed skill from this lctl binary. Local changes are preserved unless --force is supplied.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := newAISkillManager()
		if err != nil {
			return err
		}
		report, changed, err := manager.Update(aiUpdateForce)
		if err != nil {
			return err
		}
		return writeAIResult(cmd, aiCommandResult{
			Action:  "update",
			Changed: changed,
			Report:  report,
		})
	},
}

var aiUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove an lctl-managed AI skill installation",
	Long:  "Remove only the operate-launchctl skill installed by lctl. Local changes are preserved unless --force is supplied.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := newAISkillManager()
		if err != nil {
			return err
		}
		report, changed, err := manager.Uninstall(aiUninstallForce)
		if err != nil {
			return err
		}
		return writeAIResult(cmd, aiCommandResult{
			Action:  "uninstall",
			Changed: changed,
			Report:  report,
		})
	},
}

func newAISkillManager() (*aiskill.Manager, error) {
	manager, err := aiskill.New(aiCodexHome, Version)
	if err != nil {
		return nil, fmt.Errorf("initialize AI skill manager: %w", err)
	}
	return manager, nil
}

func writeAIResult(cmd *cobra.Command, result aiCommandResult) error {
	if jsonOutput {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	writer := cmd.OutOrStdout()
	switch result.Action {
	case "install":
		if result.Changed {
			fmt.Fprintf(writer, "Installed %s at %s\n", aiskill.SkillName, result.Path)
			fmt.Fprintf(writer, "Restart Codex if needed, then invoke $%s.\n", aiskill.SkillName)
		} else {
			fmt.Fprintf(writer, "%s is already installed and healthy at %s\n", aiskill.SkillName, result.Path)
		}
	case "update":
		if result.Changed {
			fmt.Fprintf(writer, "Updated %s to %s at %s\n", aiskill.SkillName, result.BundledVersion, result.Path)
		} else {
			fmt.Fprintf(writer, "%s is already current at %s\n", aiskill.SkillName, result.Path)
		}
	case "uninstall":
		if result.Changed {
			fmt.Fprintf(writer, "Removed %s from %s\n", aiskill.SkillName, result.Path)
		} else {
			fmt.Fprintf(writer, "%s is not installed at %s\n", aiskill.SkillName, result.Path)
		}
	case "doctor":
		fmt.Fprintf(writer, "Skill: %s\n", aiskill.SkillName)
		fmt.Fprintf(writer, "Status: %s\n", result.Status)
		fmt.Fprintf(writer, "Path: %s\n", result.Path)
		if result.InstalledVersion != "" {
			fmt.Fprintf(writer, "Installed version: %s\n", result.InstalledVersion)
		}
		fmt.Fprintf(writer, "Bundled version: %s\n", result.BundledVersion)
		if result.CodexCLI {
			fmt.Fprintf(writer, "Codex CLI: %s\n", result.CodexPath)
		} else {
			fmt.Fprintln(writer, "Codex CLI: not found (the desktop app can still discover local skills)")
		}
		if len(result.Changes) > 0 {
			fmt.Fprintf(writer, "Changes: %s\n", strings.Join(result.Changes, "; "))
		}
	}
	return nil
}

func init() {
	aiCmd.PersistentFlags().StringVar(&aiCodexHome, "codex-home", "", "Override the Codex home directory")
	aiUpdateCmd.Flags().BoolVar(&aiUpdateForce, "force", false, "Replace local changes in the managed skill")
	aiUninstallCmd.Flags().BoolVar(&aiUninstallForce, "force", false, "Remove a managed skill even when it has local changes")

	aiCmd.AddCommand(aiInstallCmd)
	aiCmd.AddCommand(aiDoctorCmd)
	aiCmd.AddCommand(aiUpdateCmd)
	aiCmd.AddCommand(aiUninstallCmd)
	rootCmd.AddCommand(aiCmd)
}
