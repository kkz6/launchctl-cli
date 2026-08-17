package cmd

import (
	"path/filepath"
	"testing"

	"github.com/kkz6/launchctl/internal/selfupdate"
)

func TestCachedAvailableUpdate(t *testing.T) {
	manager := &selfupdate.Manager{Cache: selfupdate.Cache{
		Path: filepath.Join(t.TempDir(), "update.json"),
	}}
	if err := manager.Cache.RecordSuccess(selfupdate.Release{
		Version: "1.2.0",
		Source:  "test",
	}); err != nil {
		t.Fatal(err)
	}

	if got := cachedAvailableUpdate(manager, "1.1.0"); got != "1.2.0" {
		t.Fatalf("cachedAvailableUpdate() = %q, want 1.2.0", got)
	}
	for _, current := range []string{"1.2.0", "1.3.0", "dev"} {
		if got := cachedAvailableUpdate(manager, current); got != "" {
			t.Errorf("cachedAvailableUpdate(%q) = %q, want empty", current, got)
		}
	}
	if got := cachedAvailableUpdate(nil, "1.1.0"); got != "" {
		t.Fatalf("cachedAvailableUpdate(nil) = %q, want empty", got)
	}
}
