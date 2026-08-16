package aiskill

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultCodexHomeUsesEnvironment(t *testing.T) {
	home := filepath.Join(t.TempDir(), "custom-codex")
	t.Setenv("CODEX_HOME", home)

	got, err := DefaultCodexHome()
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(home)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("DefaultCodexHome() = %q, want %q", got, want)
	}
}

func TestInstallIsAtomicAndIdempotent(t *testing.T) {
	manager := testManager(t, t.TempDir(), "0.2.0")

	report, changed, err := manager.Install()
	if err != nil {
		t.Fatal(err)
	}
	if !changed || report.Status != StatusHealthy || report.InstalledVersion != "0.2.0" {
		t.Fatalf("unexpected install result: changed=%v report=%+v", changed, report)
	}

	for _, name := range []string{"SKILL.md", "agents/openai.yaml", "references/commands.md", markerName} {
		info, err := os.Stat(filepath.Join(manager.Path(), filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("missing installed file %s: %v", name, err)
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("installed path %s is not a regular file", name)
		}
	}

	report, changed, err = manager.Install()
	if err != nil {
		t.Fatal(err)
	}
	if changed || report.Status != StatusHealthy {
		t.Fatalf("second install should be a no-op: changed=%v report=%+v", changed, report)
	}
}

func TestUpdateTracksVersions(t *testing.T) {
	home := t.TempDir()
	oldManager := testManager(t, home, "0.1.0")
	if _, _, err := oldManager.Install(); err != nil {
		t.Fatal(err)
	}

	manager := testManager(t, home, "0.2.0")
	report, err := manager.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusUpdateAvailable || report.InstalledVersion != "0.1.0" {
		t.Fatalf("unexpected pre-update report: %+v", report)
	}

	report, changed, err := manager.Update(false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || report.Status != StatusHealthy || report.InstalledVersion != "0.2.0" {
		t.Fatalf("unexpected update result: changed=%v report=%+v", changed, report)
	}
}

func TestModifiedSkillRequiresForceToUpdate(t *testing.T) {
	manager := testManager(t, t.TempDir(), "0.2.0")
	if _, _, err := manager.Install(); err != nil {
		t.Fatal(err)
	}

	skillPath := filepath.Join(manager.Path(), "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("locally modified\n"), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := manager.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusModified || len(report.Changes) == 0 {
		t.Fatalf("modified skill was not detected: %+v", report)
	}
	if _, _, err := manager.Update(false); err == nil {
		t.Fatal("update replaced local changes without --force")
	}

	report, changed, err := manager.Update(true)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || report.Status != StatusHealthy {
		t.Fatalf("forced update failed: changed=%v report=%+v", changed, report)
	}
	want, err := fs.ReadFile(manager.bundle, "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("forced update did not restore the bundled skill")
	}
}

func TestUninstallPreservesModifiedAndUnmanagedSkills(t *testing.T) {
	manager := testManager(t, t.TempDir(), "0.2.0")
	if _, _, err := manager.Install(); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(manager.Path(), "custom.md")
	if err := os.WriteFile(custom, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.Uninstall(false); err == nil {
		t.Fatal("uninstall removed a modified skill without --force")
	}
	if _, err := os.Stat(custom); err != nil {
		t.Fatalf("modified skill was not preserved: %v", err)
	}
	if report, changed, err := manager.Uninstall(true); err != nil || !changed || report.Status != StatusNotInstalled {
		t.Fatalf("forced uninstall failed: changed=%v report=%+v err=%v", changed, report, err)
	}

	unmanaged := testManager(t, t.TempDir(), "0.2.0")
	if err := os.MkdirAll(unmanaged.Path(), 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(unmanaged.Path(), "SKILL.md")
	if err := os.WriteFile(file, []byte("user-owned"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := unmanaged.Install(); err == nil {
		t.Fatal("install replaced an unmanaged skill")
	}
	if _, _, err := unmanaged.Uninstall(true); err == nil {
		t.Fatal("uninstall --force removed an unmanaged skill")
	}
	if got, err := os.ReadFile(file); err != nil || string(got) != "user-owned" {
		t.Fatalf("unmanaged skill changed: contents=%q err=%v", got, err)
	}
}

func TestInspectRejectsSymlinkAndUnsafeMarker(t *testing.T) {
	manager := testManager(t, t.TempDir(), "0.2.0")
	if err := os.MkdirAll(filepath.Dir(manager.Path()), 0755); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	if err := os.Symlink(target, manager.Path()); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	report, err := manager.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusInvalid {
		t.Fatalf("symlink status = %s, want %s", report.Status, StatusInvalid)
	}

	unsafe := testManager(t, t.TempDir(), "0.2.0")
	if err := os.MkdirAll(unsafe.Path(), 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(marker{
		SchemaVersion: markerSchema,
		Skill:         SkillName,
		Version:       "0.2.0",
		Files:         map[string]string{"../outside": strings.Repeat("0", 64)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unsafe.Path(), markerName), data, 0644); err != nil {
		t.Fatal(err)
	}
	report, err = unsafe.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusInvalid {
		t.Fatalf("unsafe marker status = %s, want %s", report.Status, StatusInvalid)
	}
}

func TestUninstallIsIdempotent(t *testing.T) {
	manager := testManager(t, t.TempDir(), "0.2.0")
	report, changed, err := manager.Uninstall(false)
	if err != nil {
		t.Fatal(err)
	}
	if changed || report.Status != StatusNotInstalled {
		t.Fatalf("unexpected uninstall result: changed=%v report=%+v", changed, report)
	}
}

func testManager(t *testing.T, home, version string) *Manager {
	t.Helper()
	manager, err := New(home, version)
	if err != nil {
		t.Fatal(err)
	}
	return manager
}
