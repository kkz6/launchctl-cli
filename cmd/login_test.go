package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kkz6/launchctl/internal/api"
	"github.com/kkz6/launchctl/internal/config"
	"github.com/spf13/cobra"
)

func deviceLoginTestCommand() (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	command := newBrowserLoginCommand("auth")
	command.SetContext(context.Background())
	command.Flags().Bool("json", true, "")
	command.Flags().Bool("ci", false, "")
	out, diagnostics := &bytes.Buffer{}, &bytes.Buffer{}
	command.SetOut(out)
	command.SetErr(diagnostics)
	return command, out, diagnostics
}

func TestBrowserLoginApprovesAndSavesOnlySelectedProfile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var polls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/device/") && (r.Header.Get("Authorization") != "" || r.Header.Get("X-Team-ID") != "") {
			t.Error("device flow sent old credentials")
		}
		switch r.URL.Path {
		case "/auth/device/code":
			if r.Method != http.MethodPost {
				t.Errorf("method %s", r.Method)
			}
			fmt.Fprint(w, `{"success":true,"data":{"device_code":"private-device-secret","user_code":"12345678","verification_uri":"https://launchctl.io/cli/auth","expires_in":600,"interval":5}}`)
		case "/auth/device/token":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["device_code"] != "private-device-secret" {
				t.Errorf("poll body %v, error %v", body, err)
			}
			polls++
			if polls == 1 {
				fmt.Fprint(w, `{"success":true,"data":{"status":"pending"}}`)
				return
			}
			if polls == 2 {
				fmt.Fprint(w, `{"success":true,"data":{"status":"slow_down"}}`)
				return
			}
			fmt.Fprint(w, `{"success":true,"data":{"status":"approved","access_token":"private-issued-credential"}}`)
		case "/auth/user":
			if r.Header.Get("Authorization") != "Bearer private-issued-credential" {
				t.Error("identity was not verified with issued credential")
			}
			fmt.Fprint(w, `{"success":true,"data":{"id":"user-1","name":"Test User","email":"user@example.test","current_team":{"id":"team-new","name":"New team"}}}`)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	configuration := &config.Config{APIURL: server.URL, ActiveProfile: "work", AccessToken: "old-credential", TeamID: "team-old", Profiles: map[string]*config.Profile{"work": {AccessToken: "old-credential"}, "other": {AccessToken: "untouched"}}}
	if err := configuration.Save(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(os.Getenv("HOME"), ".config/launchctl/config.json")
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	command, out, diagnostics := deviceLoginTestCommand()
	var opened string
	var waits []time.Duration
	err := browserLogin(command, configuration, false, func(ctx context.Context, uri string) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 5*time.Second {
			t.Error("browser launch was not bounded independently of sign-in")
		}
		opened = uri
		return nil
	}, func(ctx context.Context, duration time.Duration) error {
		waits = append(waits, duration)
		return ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	if opened != "https://launchctl.io/cli/auth" || fmt.Sprint(waits) != "[5s 5s 10s]" {
		t.Fatalf("opened %q, waits %v", opened, waits)
	}
	var result struct {
		Authenticated bool             `json:"authenticated"`
		User          api.UserResponse `json:"user"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || !result.Authenticated || result.User.ID != "user-1" {
		t.Fatalf("output %q, error %v", out.String(), err)
	}
	combined := out.String() + diagnostics.String()
	if strings.Contains(combined, "private-") || !strings.Contains(diagnostics.String(), "12345678") {
		t.Fatalf("unsafe/missing output %q", combined)
	}
	saved, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.AccessToken != "private-issued-credential" || saved.TeamID != "team-new" || saved.Profiles["other"].AccessToken != "untouched" {
		t.Fatal("saved identity/profile mismatch")
	}
	stat, err := os.Stat(path)
	if err != nil || stat.Mode().Perm() != 0600 {
		t.Fatalf("unsafe config permissions: %v %v", stat, err)
	}
}

func TestBrowserLoginDenialAndExpiryPreserveExistingSignIn(t *testing.T) {
	for _, status := range []string{"denied", "expired", "unexpected"} {
		t.Run(status, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/code") {
					fmt.Fprint(w, `{"success":true,"data":{"device_code":"secret","user_code":"12345678","verification_uri":"https://launchctl.io/cli/auth","expires_in":600,"interval":5}}`)
				} else {
					fmt.Fprintf(w, `{"success":true,"data":{"status":%q}}`, status)
				}
			}))
			defer server.Close()
			configuration := &config.Config{APIURL: server.URL, AccessToken: "old"}
			if err := configuration.Save(); err != nil {
				t.Fatal(err)
			}
			command, out, _ := deviceLoginTestCommand()
			err := browserLogin(command, configuration, true, func(context.Context, string) error { t.Fatal("--no-browser opened browser"); return nil }, func(ctx context.Context, _ time.Duration) error { return ctx.Err() })
			if err == nil || out.Len() != 0 {
				t.Fatalf("error=%v, output=%q", err, out.String())
			}
			saved, err := config.Load()
			if err != nil || saved.AccessToken != "old" {
				t.Fatal("failed login replaced existing sign-in")
			}
		})
	}
}

func TestBrowserLoginFallsBackToPrintedURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"data":{"device_code":"secret","user_code":"12345678","verification_uri":"https://launchctl.io/cli/auth","expires_in":600,"interval":5}}`)
	}))
	defer server.Close()
	command, _, diagnostics := deviceLoginTestCommand()
	err := browserLogin(command, &config.Config{APIURL: server.URL}, false, func(context.Context, string) error { return errors.New("no desktop") }, func(context.Context, time.Duration) error { return context.Canceled })
	if !errors.Is(err, context.Canceled) || !strings.Contains(diagnostics.String(), "Open the URL above") {
		t.Fatalf("error=%v, diagnostics=%s", err, diagnostics)
	}
}

func TestBrowserLoginRejectsCIBeforeNetwork(t *testing.T) {
	command, out, diagnostics := deviceLoginTestCommand()
	if err := command.Flags().Set("ci", "true"); err != nil {
		t.Fatal(err)
	}
	err := browserLogin(command, &config.Config{APIURL: "http://127.0.0.1:1"}, false, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "--ci") || out.Len()+diagnostics.Len() != 0 {
		t.Fatalf("error=%v", err)
	}
}

func TestValidateDeviceAuthorization(t *testing.T) {
	for _, uri := range []string{"javascript:alert(1)", "file:///etc/passwd", "http://example.test", "https://user:password@example.test", "https://example.test/?device_code=secret"} {
		value := &api.DeviceAuthorization{DeviceCode: "secret", UserCode: "12345678", VerificationURI: uri, ExpiresIn: 600, Interval: 5}
		if err := validateDeviceAuthorization(value); err == nil {
			t.Errorf("accepted %s", uri)
		}
	}
	for _, uri := range []string{"https://launchctl.io/cli/auth", "http://localhost:3000/cli/auth"} {
		value := &api.DeviceAuthorization{DeviceCode: "secret", UserCode: "12345678", VerificationURI: uri, ExpiresIn: 600, Interval: 5}
		if err := validateDeviceAuthorization(value); err != nil {
			t.Errorf("rejected %s: %v", uri, err)
		}
	}
}

func TestLoginPollRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitForLoginPoll(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}
