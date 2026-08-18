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

func TestEmbeddedSkillDocumentsTypedDockerApplicationWorkflows(t *testing.T) {
	data, err := fs.ReadFile(SkillFS, "skills/operate-launchctl/references/docker-applications.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"lctl docker projects create",
		"lctl docker projects list",
		"lctl docker applications create",
		"lctl docker applications show",
		"lctl docker applications deploy",
		"lctl docker applications reload",
		"lctl docker applications deployments",
		"docker.application.deploying",
		"docker.application.deployed",
		"docker.application.failed",
		"--remove-volumes",
		"current runtime environment",
		"does not create a deployment-history",
		"/gha/resync",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Docker application reference is missing %q", want)
		}
	}

	for _, stale := range []string{
		"current CLI has no typed Docker command group",
		"remove_volumes=true",
		"reload only restarts",
	} {
		if strings.Contains(text, stale) {
			t.Fatalf("Docker application reference contains stale guidance %q", stale)
		}
	}
}

func TestEmbeddedCommandMapPrefersTypedDockerCommands(t *testing.T) {
	data, err := fs.ReadFile(SkillFS, "skills/operate-launchctl/references/commands.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"lctl docker projects list",
		"lctl docker applications show",
		"lctl docker applications deploy",
		"--remove-volumes",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("command map is missing typed Docker command %q", want)
		}
	}
	if strings.Contains(text, "There is no dedicated typed Docker command group") {
		t.Fatal("command map still says typed Docker commands are unavailable")
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

func TestEmbeddedSkillRoutesDockerWorkToTypedCommands(t *testing.T) {
	data, err := fs.ReadFile(SkillFS, "skills/operate-launchctl/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"lctl docker projects",
		"lctl docker applications",
		"--remove-volumes",
		"recreates the container with its current environment and configuration",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("SKILL.md is missing typed Docker guidance %q", want)
		}
	}
	if strings.Contains(text, "Docker workloads currently use confirmed API routes") {
		t.Fatal("SKILL.md still routes core Docker work through the raw API")
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
