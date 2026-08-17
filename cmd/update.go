package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/kkz6/launchctl/internal/selfupdate"
	"github.com/spf13/cobra"
)

var (
	updateCheck      bool
	updateForce      bool
	updateQuiet      bool
	updateBackground bool

	newSelfUpdateManager = selfupdate.NewManager
)

type updateCommandResult struct {
	Action string `json:"action"`
	selfupdate.UpdateResult
}

var updateCmd = &cobra.Command{
	Use:         "update",
	Aliases:     []string{"upgrade"},
	Short:       "Check for and install lctl updates",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if updateBackground {
			updateCheck = true
			updateQuiet = true
		}
		if updateCheck && updateForce {
			return errors.New("--check and --force cannot be used together")
		}

		manager, err := newSelfUpdateManager()
		if err != nil {
			return fmt.Errorf("initialize updater: %w", err)
		}
		if updateBackground {
			defer manager.Cache.ReleaseRefreshLock()
		}

		if updateCheck {
			status, err := manager.Check(cmd.Context(), Version, true)
			if err != nil {
				return err
			}
			return writeUpdateResult(cmd, updateCommandResult{
				Action:       "check",
				UpdateResult: selfupdate.UpdateResult{Status: status},
			})
		}

		stdout, stderr := io.Writer(cmd.OutOrStdout()), io.Writer(cmd.ErrOrStderr())
		if jsonOutput {
			// Package-manager output would corrupt the single JSON result.
			stdout, stderr = io.Discard, io.Discard
		}
		result, err := manager.Update(cmd.Context(), Version, updateForce, stdout, stderr)
		if err != nil {
			return err
		}
		return writeUpdateResult(cmd, updateCommandResult{Action: "update", UpdateResult: result})
	},
}

func writeUpdateResult(cmd *cobra.Command, result updateCommandResult) error {
	if updateQuiet {
		return nil
	}
	if jsonOutput {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	writer := cmd.OutOrStdout()
	switch result.Action {
	case "check":
		if result.UpdateAvailable {
			fmt.Fprintf(writer, "Update available: lctl v%s → v%s\n", result.CurrentVersion, result.LatestVersion)
			fmt.Fprintln(writer, "Run lctl update to install it.")
		} else {
			fmt.Fprintf(writer, "lctl v%s is up to date.\n", result.CurrentVersion)
		}
	case "update":
		if result.Updated {
			fmt.Fprintf(writer, "Updated lctl from v%s to v%s using %s.\n", result.CurrentVersion, result.LatestVersion, result.Method)
			fmt.Fprintln(writer, "If lctl manages your AI skill, run lctl ai update next.")
		} else {
			fmt.Fprintf(writer, "lctl v%s is already up to date.\n", result.CurrentVersion)
		}
	}
	return nil
}

func startBackgroundUpdateCheck(manager *selfupdate.Manager, current string) {
	if manager == nil || current == "dev" || strings.TrimSpace(os.Getenv("LAUNCHCTL_NO_UPDATE_CHECK")) != "" {
		return
	}
	if !manager.Cache.NeedsRefresh() {
		return
	}

	releaseLock, acquired, err := manager.Cache.TryRefreshLock()
	if err != nil || !acquired {
		return
	}
	executable, err := os.Executable()
	if err != nil {
		releaseLock()
		return
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		releaseLock()
		return
	}
	defer devNull.Close()

	command := exec.Command(executable, "update", "--check", "--quiet", "--background")
	command.Stdin = devNull
	command.Stdout = devNull
	command.Stderr = devNull
	command.Env = append(os.Environ(), "NO_COLOR=1", "TERM=dumb")
	if err := command.Start(); err != nil {
		releaseLock()
		return
	}
	_ = command.Process.Release()
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "Check for an update without installing it")
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "Reinstall the latest version even when already current")
	updateCmd.Flags().BoolVar(&updateQuiet, "quiet", false, "Suppress successful output")
	updateCmd.Flags().BoolVar(&updateBackground, "background", false, "Run an automatic background update check")
	_ = updateCmd.Flags().MarkHidden("background")
	_ = updateCmd.Flags().MarkHidden("quiet")
	rootCmd.AddCommand(updateCmd)
}
