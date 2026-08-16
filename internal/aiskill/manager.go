package aiskill

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	launchctlplugin "github.com/kkz6/launchctl/plugins/launchctl"
)

const (
	SkillName    = "operate-launchctl"
	markerName   = ".lctl-skill.json"
	markerSchema = 1
)

type Status string

const (
	StatusNotInstalled    Status = "not-installed"
	StatusHealthy         Status = "healthy"
	StatusUpdateAvailable Status = "update-available"
	StatusModified        Status = "modified"
	StatusUnmanaged       Status = "unmanaged"
	StatusInvalid         Status = "invalid"
)

type Report struct {
	Status           Status   `json:"status"`
	Path             string   `json:"path"`
	InstalledVersion string   `json:"installed_version,omitempty"`
	BundledVersion   string   `json:"bundled_version"`
	Changes          []string `json:"changes,omitempty"`
}

type marker struct {
	SchemaVersion int               `json:"schema_version"`
	Skill         string            `json:"skill"`
	Version       string            `json:"version"`
	Files         map[string]string `json:"files"`
}

type Manager struct {
	codexHome string
	dest      string
	version   string
	bundle    fs.FS
}

func DefaultCodexHome() (string, error) {
	if value := strings.TrimSpace(os.Getenv("CODEX_HOME")); value != "" {
		return filepath.Abs(value)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}

	return filepath.Join(home, ".codex"), nil
}

func New(codexHome, version string) (*Manager, error) {
	var err error
	if strings.TrimSpace(codexHome) == "" {
		codexHome, err = DefaultCodexHome()
		if err != nil {
			return nil, err
		}
	}

	codexHome, err = filepath.Abs(filepath.Clean(codexHome))
	if err != nil {
		return nil, fmt.Errorf("resolve Codex home: %w", err)
	}
	if strings.TrimSpace(version) == "" {
		version = "dev"
	}

	bundle, err := fs.Sub(launchctlplugin.SkillFS, "skills/"+SkillName)
	if err != nil {
		return nil, fmt.Errorf("open embedded skill: %w", err)
	}

	dest := filepath.Join(codexHome, "skills", SkillName)
	if filepath.Base(dest) != SkillName || filepath.Dir(dest) != filepath.Join(codexHome, "skills") {
		return nil, fmt.Errorf("unsafe skill destination %q", dest)
	}

	return &Manager{
		codexHome: codexHome,
		dest:      dest,
		version:   version,
		bundle:    bundle,
	}, nil
}

func (m *Manager) Path() string {
	return m.dest
}

func (m *Manager) Inspect() (Report, error) {
	report := Report{Path: m.dest, BundledVersion: m.version}

	info, err := os.Lstat(m.dest)
	if err != nil {
		if os.IsNotExist(err) {
			report.Status = StatusNotInstalled
			return report, nil
		}
		return report, fmt.Errorf("inspect skill destination: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		report.Status = StatusInvalid
		report.Changes = []string{"skill destination is not a regular directory"}
		return report, nil
	}

	installed, err := m.readMarker()
	if err != nil {
		if os.IsNotExist(err) {
			report.Status = StatusUnmanaged
			report.Changes = []string{"installation marker is missing"}
			return report, nil
		}
		report.Status = StatusInvalid
		report.Changes = []string{err.Error()}
		return report, nil
	}
	report.InstalledVersion = installed.Version

	if installed.SchemaVersion != markerSchema || installed.Skill != SkillName || len(installed.Files) == 0 {
		report.Status = StatusInvalid
		report.Changes = []string{"installation marker is invalid"}
		return report, nil
	}
	for name := range installed.Files {
		if !validRelativeFile(name) {
			report.Status = StatusInvalid
			report.Changes = []string{"installation marker contains an unsafe path"}
			return report, nil
		}
	}

	changes, err := m.installedChanges(installed.Files)
	if err != nil {
		return report, err
	}
	if len(changes) > 0 {
		report.Status = StatusModified
		report.Changes = changes
		return report, nil
	}

	bundled, err := hashFS(m.bundle)
	if err != nil {
		return report, err
	}
	if installed.Version != m.version || !sameHashes(installed.Files, bundled) {
		report.Status = StatusUpdateAvailable
		return report, nil
	}

	report.Status = StatusHealthy
	return report, nil
}

func (m *Manager) Install() (Report, bool, error) {
	report, err := m.Inspect()
	if err != nil {
		return report, false, err
	}

	switch report.Status {
	case StatusHealthy:
		return report, false, nil
	case StatusNotInstalled:
		if err := m.replace(false); err != nil {
			return report, false, err
		}
		installed, err := m.Inspect()
		return installed, true, err
	case StatusUpdateAvailable:
		return report, false, fmt.Errorf("%s is already installed; run lctl ai update", SkillName)
	case StatusModified:
		return report, false, fmt.Errorf("%s contains local changes; run lctl ai update --force to replace them", SkillName)
	case StatusUnmanaged, StatusInvalid:
		return report, false, fmt.Errorf("refusing to replace %s installation at %s", report.Status, report.Path)
	default:
		return report, false, fmt.Errorf("unexpected skill status %q", report.Status)
	}
}

func (m *Manager) Update(force bool) (Report, bool, error) {
	report, err := m.Inspect()
	if err != nil {
		return report, false, err
	}

	switch report.Status {
	case StatusHealthy:
		return report, false, nil
	case StatusUpdateAvailable:
		// Continue below.
	case StatusModified:
		if !force {
			return report, false, fmt.Errorf("%s contains local changes; rerun with --force to replace them", SkillName)
		}
	case StatusNotInstalled:
		return report, false, fmt.Errorf("%s is not installed; run lctl ai install", SkillName)
	case StatusUnmanaged, StatusInvalid:
		return report, false, fmt.Errorf("refusing to update %s installation at %s", report.Status, report.Path)
	default:
		return report, false, fmt.Errorf("unexpected skill status %q", report.Status)
	}

	if err := m.replace(true); err != nil {
		return report, false, err
	}
	updated, err := m.Inspect()
	return updated, true, err
}

func (m *Manager) Uninstall(force bool) (Report, bool, error) {
	report, err := m.Inspect()
	if err != nil {
		return report, false, err
	}

	switch report.Status {
	case StatusNotInstalled:
		return report, false, nil
	case StatusHealthy, StatusUpdateAvailable:
		// Safe to remove below.
	case StatusModified:
		if !force {
			return report, false, fmt.Errorf("%s contains local changes; rerun with --force to remove them", SkillName)
		}
	case StatusUnmanaged, StatusInvalid:
		return report, false, fmt.Errorf("refusing to remove %s installation at %s", report.Status, report.Path)
	default:
		return report, false, fmt.Errorf("unexpected skill status %q", report.Status)
	}

	if err := m.removeAtomically(); err != nil {
		return report, false, err
	}
	removed, err := m.Inspect()
	return removed, true, err
}

func (m *Manager) readMarker() (marker, error) {
	data, err := os.ReadFile(filepath.Join(m.dest, markerName))
	if err != nil {
		return marker{}, err
	}

	var value marker
	if err := json.Unmarshal(data, &value); err != nil {
		return marker{}, fmt.Errorf("parse installation marker: %w", err)
	}
	return value, nil
}

func (m *Manager) installedChanges(files map[string]string) ([]string, error) {
	changes := make([]string, 0)

	for name, expected := range files {
		path := filepath.Join(m.dest, filepath.FromSlash(name))
		info, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				changes = append(changes, name+" is missing")
				continue
			}
			return nil, fmt.Errorf("inspect installed file %s: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			changes = append(changes, name+" is not a regular file")
			continue
		}

		actual, err := hashFile(path)
		if err != nil {
			return nil, err
		}
		if actual != expected {
			changes = append(changes, name+" was modified")
		}
	}

	err := filepath.WalkDir(m.dest, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == m.dest || entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(m.dest, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == markerName {
			return nil
		}
		if _, ok := files[rel]; !ok {
			changes = append(changes, rel+" is untracked")
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inspect installed skill: %w", err)
	}

	sort.Strings(changes)
	return changes, nil
}

func (m *Manager) replace(existing bool) error {
	root := filepath.Dir(m.dest)
	if err := os.MkdirAll(root, 0755); err != nil {
		return fmt.Errorf("create Codex skills directory: %w", err)
	}

	stage, err := os.MkdirTemp(root, "."+SkillName+"-install-")
	if err != nil {
		return fmt.Errorf("create skill staging directory: %w", err)
	}
	defer os.RemoveAll(stage)

	if err := m.writeStage(stage); err != nil {
		return err
	}

	if !existing {
		if err := os.Rename(stage, m.dest); err != nil {
			return fmt.Errorf("install skill: %w", err)
		}
		return nil
	}

	backup, err := uniquePath(root, "."+SkillName+"-backup-")
	if err != nil {
		return err
	}
	if err := os.Rename(m.dest, backup); err != nil {
		return fmt.Errorf("stage existing skill: %w", err)
	}
	if err := os.Rename(stage, m.dest); err != nil {
		_ = os.Rename(backup, m.dest)
		return fmt.Errorf("activate updated skill: %w", err)
	}
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("remove skill backup %s: %w", backup, err)
	}
	return nil
}

func (m *Manager) writeStage(stage string) error {
	hashes, err := hashFS(m.bundle)
	if err != nil {
		return err
	}

	err = fs.WalkDir(m.bundle, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		target := filepath.Join(stage, filepath.FromSlash(name))
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("embedded skill entry %s is not a regular file", name)
		}
		data, err := fs.ReadFile(m.bundle, name)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		return fmt.Errorf("write embedded skill: %w", err)
	}

	installed := marker{
		SchemaVersion: markerSchema,
		Skill:         SkillName,
		Version:       m.version,
		Files:         hashes,
	}
	data, err := json.MarshalIndent(installed, "", "  ")
	if err != nil {
		return fmt.Errorf("encode installation marker: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(stage, markerName), data, 0644); err != nil {
		return fmt.Errorf("write installation marker: %w", err)
	}
	return nil
}

func (m *Manager) removeAtomically() error {
	root := filepath.Dir(m.dest)
	tombstone, err := uniquePath(root, "."+SkillName+"-remove-")
	if err != nil {
		return err
	}
	if err := os.Rename(m.dest, tombstone); err != nil {
		return fmt.Errorf("stage skill removal: %w", err)
	}
	if err := os.RemoveAll(tombstone); err != nil {
		_ = os.Rename(tombstone, m.dest)
		return fmt.Errorf("remove skill: %w", err)
	}
	return nil
}

func uniquePath(root, pattern string) (string, error) {
	path, err := os.MkdirTemp(root, pattern)
	if err != nil {
		return "", fmt.Errorf("reserve temporary path: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("prepare temporary path: %w", err)
	}
	return path, nil
}

func hashFS(source fs.FS) (map[string]string, error) {
	hashes := make(map[string]string)
	err := fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() || !validRelativeFile(name) {
			return fmt.Errorf("embedded skill contains invalid file %q", name)
		}
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return err
		}
		hashes[name] = hashBytes(data)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("hash embedded skill: %w", err)
	}
	if len(hashes) == 0 {
		return nil, fmt.Errorf("embedded skill is empty")
	}
	return hashes, nil
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read installed file %s: %w", path, err)
	}
	return hashBytes(data), nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sameHashes(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for name, value := range left {
		if right[name] != value {
			return false
		}
	}
	return true
}

func validRelativeFile(name string) bool {
	return name != "." && name != markerName && fs.ValidPath(name)
}
