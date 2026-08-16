package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kkz6/launchctl/internal/aiskill"
	"github.com/spf13/cobra"
)

func TestWriteAIResultJSON(t *testing.T) {
	previous := jsonOutput
	jsonOutput = true
	defer func() { jsonOutput = previous }()

	var output bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&output)
	result := aiCommandResult{
		Action:  "doctor",
		Changed: false,
		Report: aiskill.Report{
			Status:         aiskill.StatusHealthy,
			Path:           "/tmp/codex/skills/operate-launchctl",
			BundledVersion: "0.2.0",
		},
	}
	if err := writeAIResult(command, result); err != nil {
		t.Fatal(err)
	}

	var decoded aiCommandResult
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, output.String())
	}
	if decoded.Action != "doctor" || decoded.Status != aiskill.StatusHealthy {
		t.Fatalf("unexpected JSON output: %+v", decoded)
	}
}

func TestWriteAIDoctorText(t *testing.T) {
	previous := jsonOutput
	jsonOutput = false
	defer func() { jsonOutput = previous }()

	var output bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&output)
	result := aiCommandResult{
		Action: "doctor",
		Report: aiskill.Report{
			Status:         aiskill.StatusModified,
			Path:           "/tmp/codex/skills/operate-launchctl",
			BundledVersion: "0.2.0",
			Changes:        []string{"SKILL.md was modified"},
		},
	}
	if err := writeAIResult(command, result); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"Status: modified", "Codex CLI: not found", "SKILL.md was modified"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("doctor output missing %q:\n%s", want, output.String())
		}
	}
}
