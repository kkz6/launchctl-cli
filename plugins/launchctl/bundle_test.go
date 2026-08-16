package launchctlplugin

import (
	"io/fs"
	"strings"
	"testing"
)

func TestEmbeddedSkillContainsRequiredFiles(t *testing.T) {
	for _, name := range []string{
		"skills/operate-launchctl/SKILL.md",
		"skills/operate-launchctl/agents/openai.yaml",
		"skills/operate-launchctl/references/commands.md",
		"skills/operate-launchctl/references/docker-applications.md",
	} {
		data, err := fs.ReadFile(SkillFS, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if len(data) == 0 {
			t.Fatalf("embedded file %s is empty", name)
		}
	}
}

func TestEmbeddedSkillDocumentsDockerApplicationDeployments(t *testing.T) {
	data, err := fs.ReadFile(SkillFS, "skills/operate-launchctl/references/docker-applications.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"source_type",
		"docker.application.deploying",
		"docker.application.deployed",
		"docker.application.failed",
		"remove_volumes=true",
		"/gha/resync",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Docker application reference is missing %q", want)
		}
	}
}

func TestSkillMetadataNamesOperateLaunchctl(t *testing.T) {
	data, err := fs.ReadFile(SkillFS, "skills/operate-launchctl/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "---\nname: operate-launchctl\n") {
		t.Fatalf("unexpected SKILL.md frontmatter:\n%s", text)
	}
	if !strings.Contains(text, "description:") {
		t.Fatal("SKILL.md is missing its trigger description")
	}
}

func TestEmbeddedSkillTreatsLaunchctlAsHostedOnly(t *testing.T) {
	data, err := fs.ReadFile(SkillFS, "skills/operate-launchctl/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"launchctl is hosted-only",
		"approved launchctl development or staging origin",
		"never invent self-hosting instructions",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("hosted-only guidance is missing %q", want)
		}
	}
}
