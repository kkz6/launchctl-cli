package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionCommandTextAndJSON(t *testing.T) {
	previousVersion, previousJSON := Version, jsonOutput
	Version = "1.2.3"
	t.Cleanup(func() {
		Version, jsonOutput = previousVersion, previousJSON
	})

	t.Run("text", func(t *testing.T) {
		jsonOutput = false
		var output bytes.Buffer
		command := &cobra.Command{}
		command.SetOut(&output)

		if err := versionCmd.RunE(command, nil); err != nil {
			t.Fatal(err)
		}
		if got, want := output.String(), "lctl 1.2.3\n"; got != want {
			t.Fatalf("version output = %q, want %q", got, want)
		}
	})

	t.Run("json", func(t *testing.T) {
		jsonOutput = true
		var output bytes.Buffer
		command := &cobra.Command{}
		command.SetOut(&output)

		if err := versionCmd.RunE(command, nil); err != nil {
			t.Fatal(err)
		}
		var decoded map[string]string
		if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
			t.Fatalf("invalid JSON output: %v\n%s", err, output.String())
		}
		if got := decoded["version"]; got != "1.2.3" {
			t.Fatalf("JSON version = %q, want 1.2.3", got)
		}
	})
}

func TestVersionCommandSkipsConfigAndRejectsArguments(t *testing.T) {
	if !commandSkipsConfig(versionCmd) {
		t.Fatal("version command does not skip account configuration")
	}
	if err := versionCmd.Args(versionCmd, []string{"unexpected"}); err == nil {
		t.Fatal("version accepted a positional argument")
	}
}
