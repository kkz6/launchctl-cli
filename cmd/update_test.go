package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kkz6/launchctl/internal/selfupdate"
	"github.com/spf13/cobra"
)

func TestWriteUpdateResultText(t *testing.T) {
	tests := []struct {
		name   string
		result updateCommandResult
		want   []string
	}{
		{
			name: "available",
			result: updateCommandResult{
				Action: "check",
				UpdateResult: selfupdate.UpdateResult{Status: selfupdate.Status{
					CurrentVersion: "0.2.2", LatestVersion: "0.3.0", UpdateAvailable: true,
				}},
			},
			want: []string{"Update available: lctl v0.2.2 → v0.3.0", "Run lctl update to install it."},
		},
		{
			name: "current",
			result: updateCommandResult{
				Action: "check",
				UpdateResult: selfupdate.UpdateResult{Status: selfupdate.Status{
					CurrentVersion: "0.2.2", LatestVersion: "0.2.2",
				}},
			},
			want: []string{"lctl v0.2.2 is up to date."},
		},
		{
			name: "updated",
			result: updateCommandResult{
				Action: "update",
				UpdateResult: selfupdate.UpdateResult{
					Status:  selfupdate.Status{CurrentVersion: "0.2.2", LatestVersion: "0.3.0"},
					Updated: true,
					Method:  "homebrew",
				},
			},
			want: []string{
				"Updated lctl from v0.2.2 to v0.3.0 using homebrew.",
				"run lctl ai update next.",
			},
		},
		{
			name: "already current",
			result: updateCommandResult{
				Action: "update",
				UpdateResult: selfupdate.UpdateResult{Status: selfupdate.Status{
					CurrentVersion: "0.2.2", LatestVersion: "0.2.2",
				}},
			},
			want: []string{"lctl v0.2.2 is already up to date."},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setUpdateOutputMode(t, false, false)
			var output bytes.Buffer
			command := &cobra.Command{}
			command.SetOut(&output)

			if err := writeUpdateResult(command, test.result); err != nil {
				t.Fatal(err)
			}
			for _, want := range test.want {
				if !strings.Contains(output.String(), want) {
					t.Fatalf("update output missing %q:\n%s", want, output.String())
				}
			}
		})
	}
}

func TestWriteUpdateResultJSON(t *testing.T) {
	setUpdateOutputMode(t, true, false)
	var output bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&output)
	result := updateCommandResult{
		Action: "check",
		UpdateResult: selfupdate.UpdateResult{Status: selfupdate.Status{
			CurrentVersion: "0.2.2", LatestVersion: "0.3.0", UpdateAvailable: true,
		}},
	}

	if err := writeUpdateResult(command, result); err != nil {
		t.Fatal(err)
	}
	var decoded updateCommandResult
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, output.String())
	}
	if decoded.Action != "check" || decoded.CurrentVersion != "0.2.2" ||
		decoded.LatestVersion != "0.3.0" || !decoded.UpdateAvailable {
		t.Fatalf("unexpected JSON output: %+v", decoded)
	}
}

func TestWriteUpdateResultQuietSuppressesOutput(t *testing.T) {
	setUpdateOutputMode(t, true, true)
	var output bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&output)

	if err := writeUpdateResult(command, updateCommandResult{
		Action: "check",
		UpdateResult: selfupdate.UpdateResult{Status: selfupdate.Status{
			CurrentVersion: "0.2.2", LatestVersion: "0.3.0", UpdateAvailable: true,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("quiet update wrote output: %q", output.String())
	}
}

func TestUpdateCommandSkipsConfigAndHidesInternalFlags(t *testing.T) {
	if !commandSkipsConfig(updateCmd) {
		t.Fatal("update command does not skip account configuration")
	}
	for _, name := range []string{"background", "quiet"} {
		flag := updateCmd.Flags().Lookup(name)
		if flag == nil || !flag.Hidden {
			t.Fatalf("update --%s should be an internal hidden flag", name)
		}
	}
	if err := updateCmd.Args(updateCmd, []string{"unexpected"}); err == nil {
		t.Fatal("update accepted a positional argument")
	}
}

func TestUpdateCommandKeepsSuccessfulJSONMachineReadable(t *testing.T) {
	digest := strings.Repeat("a", 64)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		artifactURL := server.URL + "/v1.2.3/lctl-linux-amd64.tar.gz"
		switch request.URL.Path {
		case "/latest.json":
			fmt.Fprintf(writer, `{"schema_version":1,"channel":"stable","version":"1.2.3","tag":"v1.2.3","artifacts":[{"os":"linux","arch":"amd64","filename":"lctl-linux-amd64.tar.gz","url":%q,"sha256":%q}]}`, artifactURL, digest)
		case "/formula.rb":
			fmt.Fprintf(writer, "version \"1.2.3\"\nurl %q\nsha256 %q\n", artifactURL, digest)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	manager := &selfupdate.Manager{
		Source: selfupdate.Source{
			Client: server.Client(), ManifestURL: server.URL + "/latest.json",
			FormulaURL: server.URL + "/formula.rb", GOOS: "linux", GOARCH: "amd64", AllowHTTP: true,
		},
		Cache:      selfupdate.Cache{Path: filepath.Join(t.TempDir(), "update.json")},
		Executable: func() (string, error) { return "/opt/homebrew/bin/lctl", nil },
		EvalSymlinks: func(string) (string, error) {
			return "/opt/homebrew/Cellar/lctl/1.2.3/bin/lctl", nil
		},
		LookPath: func(string) (string, error) { return "/opt/homebrew/bin/brew", nil },
		RunCommand: func(_ context.Context, _ string, _ []string, stdout, stderr io.Writer) error {
			_, _ = io.WriteString(stdout, "package-manager stdout")
			_, _ = io.WriteString(stderr, "package-manager stderr")
			return nil
		},
		ValidateBinary: func(context.Context, string, string) error { return nil },
	}

	previousManager, previousVersion := newSelfUpdateManager, Version
	previousCheck, previousForce, previousBackground := updateCheck, updateForce, updateBackground
	newSelfUpdateManager = func() (*selfupdate.Manager, error) { return manager, nil }
	Version, updateCheck, updateForce, updateBackground = "1.2.3", false, true, false
	setUpdateOutputMode(t, true, false)
	t.Cleanup(func() {
		newSelfUpdateManager, Version = previousManager, previousVersion
		updateCheck, updateForce, updateBackground = previousCheck, previousForce, previousBackground
	})

	var output, commandError bytes.Buffer
	command := &cobra.Command{}
	command.SetContext(context.Background())
	command.SetOut(&output)
	command.SetErr(&commandError)
	if err := updateCmd.RunE(command, nil); err != nil {
		t.Fatal(err)
	}
	var decoded updateCommandResult
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("update output is not one JSON object: %v\n%s", err, output.String())
	}
	if !decoded.Updated || decoded.Method != "homebrew" {
		t.Fatalf("unexpected update result: %+v", decoded)
	}
	if strings.Contains(output.String(), "package-manager") || commandError.Len() != 0 {
		t.Fatalf("package-manager output leaked into JSON: stdout=%q stderr=%q", output.String(), commandError.String())
	}
}

func TestBackgroundUpdateRejectsForceBeforeInitialization(t *testing.T) {
	previousManager := newSelfUpdateManager
	previousCheck, previousForce, previousQuiet, previousBackground := updateCheck, updateForce, updateQuiet, updateBackground
	newSelfUpdateManager = func() (*selfupdate.Manager, error) {
		t.Fatal("manager initialized for invalid flags")
		return nil, nil
	}
	updateCheck, updateForce, updateQuiet, updateBackground = false, true, false, true
	t.Cleanup(func() {
		newSelfUpdateManager = previousManager
		updateCheck, updateForce, updateQuiet, updateBackground = previousCheck, previousForce, previousQuiet, previousBackground
	})

	if err := updateCmd.RunE(&cobra.Command{}, nil); err == nil || !strings.Contains(err.Error(), "--check and --force") {
		t.Fatalf("RunE() error = %v, want flag conflict", err)
	}
}

func setUpdateOutputMode(t *testing.T, jsonMode, quietMode bool) {
	t.Helper()
	previousJSON, previousQuiet := jsonOutput, updateQuiet
	jsonOutput, updateQuiet = jsonMode, quietMode
	t.Cleanup(func() {
		jsonOutput, updateQuiet = previousJSON, previousQuiet
	})
}
