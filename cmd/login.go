package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/commandutil"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/spf13/cobra"
)

func newBrowserLoginCommand(name string) *cobra.Command {
	var noBrowser bool
	command := &cobra.Command{
		Use: name, Short: "Sign in through your browser using a one-time code",
		Args: cobra.NoArgs, SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return browserLogin(cmd, cfg, noBrowser, openLoginBrowser, waitForLoginPoll)
		},
	}
	command.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the sign-in URL without opening a browser")
	return command
}

func browserLogin(cmd *cobra.Command, configuration *config.Config, noBrowser bool, open func(context.Context, string) error, wait func(context.Context, time.Duration) error) error {
	ci, _ := cmd.Flags().GetBool("ci")
	if ci {
		return fmt.Errorf("browser sign-in needs your approval; run lctl auth without --ci")
	}
	// Starting a new sign-in must not send credentials from a previous session.
	client := api.NewClient(&config.Config{APIURL: configuration.EffectiveAPIURL()})
	requestCtx, requestCancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	authorization, err := client.StartDeviceAuthorization(requestCtx)
	requestCancel()
	if err != nil {
		return fmt.Errorf("could not start browser sign-in: %w", err)
	}
	if err := validateDeviceAuthorization(authorization); err != nil {
		return err
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Your one-time code: %s\nOpen %s, sign in, and enter this code to approve lctl.\n", authorization.UserCode, authorization.VerificationURI)
	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(authorization.ExpiresIn)*time.Second)
	defer cancel()
	if !noBrowser {
		browserCtx, browserCancel := context.WithTimeout(ctx, 5*time.Second)
		openErr := open(browserCtx, authorization.VerificationURI)
		browserCancel()
		if openErr != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Could not open a browser. Open the URL above to continue.")
		}
	}
	interval := time.Duration(authorization.Interval) * time.Second
	for {
		if err := wait(ctx, interval); err != nil {
			return fmt.Errorf("browser sign-in stopped: %w", err)
		}
		result, err := client.PollDeviceAuthorization(ctx, authorization.DeviceCode)
		if err != nil {
			return fmt.Errorf("could not finish browser sign-in: %w", err)
		}
		switch result.Status {
		case "pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
			continue
		case "denied":
			return fmt.Errorf("browser sign-in was denied")
		case "expired":
			return fmt.Errorf("the sign-in code expired; run lctl auth again")
		case "approved":
			if result.AccessToken == "" {
				return fmt.Errorf("browser sign-in returned no credential")
			}
			// Confirm identity through the issued credential before persisting it.
			identityClient := api.NewClient(&config.Config{APIURL: configuration.EffectiveAPIURL(), AccessToken: result.AccessToken})
			user, err := identityClient.GetUserContext(ctx)
			if err != nil {
				return fmt.Errorf("could not verify browser sign-in: %w", err)
			}
			if user.ID == "" {
				return fmt.Errorf("browser sign-in returned no user identity")
			}
			configuration.AccessToken = result.AccessToken
			configuration.UserID, configuration.UserName, configuration.UserEmail = user.ID, user.Name, user.Email
			configuration.TeamID, configuration.TeamName = "", ""
			if user.CurrentTeam != nil {
				configuration.TeamID, configuration.TeamName = user.CurrentTeam.ID, user.CurrentTeam.Name
			} else if user.CurrentTeamID != nil {
				configuration.TeamID = *user.CurrentTeamID
			}
			if err := configuration.Save(); err != nil {
				return fmt.Errorf("could not save sign-in: %w", err)
			}
			if commandutil.JSONEnabled(cmd) {
				return commandutil.WriteJSON(cmd, map[string]any{"authenticated": true, "user": user})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Signed in as %s (%s).\n", user.Name, user.Email)
			return err
		default:
			return fmt.Errorf("browser sign-in returned an unknown status")
		}
	}
}

func validateDeviceAuthorization(value *api.DeviceAuthorization) error {
	if value.DeviceCode == "" || len(value.UserCode) != 8 || strings.Trim(value.UserCode, "0123456789") != "" || value.ExpiresIn < 1 || value.ExpiresIn > 900 || value.Interval < 1 || value.Interval > 60 {
		return fmt.Errorf("invalid browser sign-in response")
	}
	uri, err := url.Parse(value.VerificationURI)
	if err != nil || uri.Host == "" || uri.User != nil || uri.RawQuery != "" || uri.Fragment != "" {
		return fmt.Errorf("invalid browser sign-in URL")
	}
	local := uri.Hostname() == "localhost" || uri.Hostname() == "127.0.0.1" || uri.Hostname() == "::1"
	if uri.Scheme != "https" && !(uri.Scheme == "http" && local) {
		return fmt.Errorf("browser sign-in requires an HTTPS URL")
	}
	return nil
}

func waitForLoginPoll(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func openLoginBrowser(ctx context.Context, uri string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.CommandContext(ctx, "open", uri)
	case "linux":
		command = exec.CommandContext(ctx, "xdg-open", uri)
	default:
		return fmt.Errorf("automatic browser opening is unsupported")
	}
	return command.Run()
}

func init() { rootCmd.AddCommand(newBrowserLoginCommand("login")) }
