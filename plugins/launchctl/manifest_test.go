package launchctlplugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

type pluginManifest struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      struct {
		Name string `json:"name"`
	} `json:"author"`
	Skills    string `json:"skills"`
	Interface struct {
		DisplayName      string   `json:"displayName"`
		ShortDescription string   `json:"shortDescription"`
		LongDescription  string   `json:"longDescription"`
		DeveloperName    string   `json:"developerName"`
		Category         string   `json:"category"`
		DefaultPrompt    []string `json:"defaultPrompt"`
	} `json:"interface"`
}

type marketplaceManifest struct {
	Name    string `json:"name"`
	Plugins []struct {
		Name   string `json:"name"`
		Source struct {
			Type string `json:"source"`
			Path string `json:"path"`
		} `json:"source"`
		Policy struct {
			Installation   string `json:"installation"`
			Authentication string `json:"authentication"`
		} `json:"policy"`
		Category string `json:"category"`
	} `json:"plugins"`
}

func TestPluginManifestIsDistributionReady(t *testing.T) {
	var manifest pluginManifest
	if err := json.Unmarshal(PluginManifest, &manifest); err != nil {
		t.Fatalf("parse plugin manifest: %v", err)
	}
	if manifest.Name != "launchctl" || manifest.Description == "" || manifest.Author.Name == "" {
		t.Fatalf("plugin identity is incomplete: %+v", manifest)
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(manifest.Version) {
		t.Fatalf("plugin version %q is not strict semver", manifest.Version)
	}
	if manifest.Skills != "./skills/" {
		t.Fatalf("skills path = %q, want ./skills/", manifest.Skills)
	}
	if manifest.Interface.DisplayName == "" || manifest.Interface.ShortDescription == "" ||
		manifest.Interface.LongDescription == "" || manifest.Interface.DeveloperName == "" ||
		manifest.Interface.Category == "" || len(manifest.Interface.DefaultPrompt) == 0 {
		t.Fatalf("plugin interface metadata is incomplete: %+v", manifest.Interface)
	}
}

func TestRepoMarketplacePointsToPlugin(t *testing.T) {
	path := filepath.Join("..", "..", ".agents", "plugins", "marketplace.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read marketplace: %v", err)
	}
	var marketplace marketplaceManifest
	if err := json.Unmarshal(data, &marketplace); err != nil {
		t.Fatalf("parse marketplace: %v", err)
	}
	if marketplace.Name != "launchctl" || len(marketplace.Plugins) != 1 {
		t.Fatalf("unexpected marketplace identity: %+v", marketplace)
	}
	entry := marketplace.Plugins[0]
	if entry.Name != "launchctl" || entry.Source.Type != "local" || entry.Source.Path != "./plugins/launchctl" {
		t.Fatalf("unexpected marketplace source: %+v", entry)
	}
	if entry.Policy.Installation == "" || entry.Policy.Authentication == "" || entry.Category == "" {
		t.Fatalf("marketplace policy is incomplete: %+v", entry)
	}
}
