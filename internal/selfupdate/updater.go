package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	maxArchiveBytes         = 64 << 20
	maxExpandedArchiveBytes = 192 << 20
	maxBinaryBytes          = 128 << 20
)

type CommandRunner func(context.Context, string, []string, io.Writer, io.Writer) error

type Manager struct {
	Source         Source
	Cache          Cache
	Executable     func() (string, error)
	EvalSymlinks   func(string) (string, error)
	LookPath       func(string) (string, error)
	RunCommand     CommandRunner
	ValidateBinary func(context.Context, string, string) error
}

type UpdateResult struct {
	Status
	Updated    bool   `json:"updated"`
	Method     string `json:"method,omitempty"`
	Executable string `json:"executable,omitempty"`
}

func NewManager() (*Manager, error) {
	cache, err := DefaultCache()
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	return &Manager{
		Source:       DefaultSource(client),
		Cache:        cache,
		Executable:   os.Executable,
		EvalSymlinks: filepath.EvalSymlinks,
		LookPath:     exec.LookPath,
		RunCommand:   runCommand,
		ValidateBinary: func(ctx context.Context, path, version string) error {
			return validateStagedBinary(ctx, path, version)
		},
	}, nil
}

func (m *Manager) Check(ctx context.Context, current string, force bool) (Status, error) {
	currentVersion, ok := normalizeVersion(current)
	if !ok {
		return Status{}, ErrDevelopmentBuild
	}

	if !force && !m.Cache.NeedsRefresh() {
		if cached, found, err := m.Cache.CachedStatus(currentVersion); err == nil && found {
			return cached, nil
		}
		return Status{CurrentVersion: currentVersion}, nil
	}

	release, err := m.Source.Fetch(ctx)
	if err != nil {
		_ = m.Cache.RecordFailure(err)
		return Status{}, err
	}
	// Caching is an optimization for passive notices. A permissions or disk
	// problem in the cache directory must not break an explicit network check.
	_ = m.Cache.RecordSuccess(release)
	status, err := StatusFor(currentVersion, release)
	if err != nil {
		return Status{}, err
	}
	status.CheckedAt = m.Cache.now().UTC().Format(time.RFC3339)
	return status, nil
}

func (m *Manager) CachedStatus(current string) (Status, bool) {
	status, found, err := m.Cache.CachedStatus(current)
	if err != nil {
		return Status{}, false
	}
	return status, found
}

func (m *Manager) Update(ctx context.Context, current string, force bool, stdout, stderr io.Writer) (UpdateResult, error) {
	currentVersion, ok := normalizeVersion(current)
	if !ok {
		return UpdateResult{}, ErrDevelopmentBuild
	}
	release, err := m.Source.Fetch(ctx)
	if err != nil {
		_ = m.Cache.RecordFailure(err)
		return UpdateResult{}, err
	}
	// Updating the executable must not depend on the optional notice cache.
	_ = m.Cache.RecordSuccess(release)
	status, err := StatusFor(currentVersion, release)
	if err != nil {
		return UpdateResult{}, err
	}
	status.CheckedAt = m.Cache.now().UTC().Format(time.RFC3339)
	result := UpdateResult{Status: status}
	if semver.Compare("v"+release.Version, "v"+currentVersion) < 0 {
		return result, fmt.Errorf("installed lctl v%s is newer than published v%s; refusing to downgrade", currentVersion, release.Version)
	}
	if !status.UpdateAvailable && !force {
		return result, nil
	}

	executable, resolved, method, err := m.installation()
	if err != nil {
		return UpdateResult{}, err
	}
	result.Method = method
	result.Executable = resolved

	if method == "homebrew" {
		linkedExecutable, ok := homebrewLinkedExecutable(resolved)
		if !ok {
			return UpdateResult{}, fmt.Errorf("cannot locate the linked Homebrew executable from %s", resolved)
		}
		if err := m.updateHomebrew(ctx, force, linkedExecutable, release.Version, stdout, stderr); err != nil {
			return UpdateResult{}, err
		}
		result.Updated = true
		return result, nil
	}

	if executable != resolved {
		return UpdateResult{}, fmt.Errorf("refusing to replace symlinked executable %s; update its installation manually", executable)
	}
	if err := m.updateDirect(ctx, resolved, release); err != nil {
		return UpdateResult{}, err
	}
	result.Updated = true
	return result, nil
}

func (m *Manager) installation() (executable, resolved, method string, err error) {
	if m.Executable == nil || m.EvalSymlinks == nil {
		return "", "", "", errors.New("updater executable resolver is not configured")
	}
	executable, err = m.Executable()
	if err != nil {
		return "", "", "", fmt.Errorf("locate current executable: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve current executable path: %w", err)
	}
	resolved, err = m.EvalSymlinks(executable)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve current executable symlinks: %w", err)
	}
	method = "direct"
	if isHomebrewPath(resolved) {
		method = "homebrew"
	}
	return executable, resolved, method, nil
}

func (m *Manager) updateHomebrew(ctx context.Context, force bool, linkedExecutable, version string, stdout, stderr io.Writer) error {
	if m.LookPath == nil || m.RunCommand == nil || m.ValidateBinary == nil {
		return errors.New("Homebrew updater is not configured")
	}
	brew, err := m.LookPath("brew")
	if err != nil {
		return errors.New("this lctl installation is managed by Homebrew, but brew is not available in PATH")
	}
	action := "upgrade"
	if force {
		action = "reinstall"
	}
	if err := m.RunCommand(ctx, brew, []string{action, "kkz6/tap/lctl"}, stdout, stderr); err != nil {
		return fmt.Errorf("Homebrew update failed: %w\nIf the formula is not trusted, run: brew trust --formula kkz6/tap/lctl", err)
	}
	if err := m.ValidateBinary(ctx, linkedExecutable, version); err != nil {
		return fmt.Errorf("Homebrew completed but installed lctl validation failed: %w", err)
	}
	return nil
}

func (m *Manager) updateDirect(ctx context.Context, target string, release Release) error {
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("inspect current executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("current executable %s is not a regular file", target)
	}

	// Stage beside the target so hardened systems with a noexec temporary
	// directory can still validate the candidate. This also proves directory
	// write access before downloading and executing it.
	staged, err := os.CreateTemp(filepath.Dir(target), ".lctl-candidate-*")
	if err != nil {
		return fmt.Errorf("create update staging file: %w", err)
	}
	stagedPath := staged.Name()
	if err := staged.Close(); err != nil {
		_ = os.Remove(stagedPath)
		return fmt.Errorf("close update staging file: %w", err)
	}
	defer os.Remove(stagedPath)

	archivePath, err := m.downloadArchive(ctx, release.Artifact)
	if err != nil {
		return err
	}
	defer os.Remove(archivePath)

	if err := extractBinary(archivePath, stagedPath); err != nil {
		return err
	}
	if err := os.Chmod(stagedPath, 0700); err != nil {
		return fmt.Errorf("make staged lctl executable: %w", err)
	}
	if m.ValidateBinary == nil {
		return errors.New("updater binary validator is not configured")
	}
	if err := m.ValidateBinary(ctx, stagedPath, release.Version); err != nil {
		return err
	}

	if err := replaceExecutable(target, stagedPath, info.Mode().Perm()); err != nil {
		return err
	}
	return nil
}

func (m *Manager) downloadArchive(ctx context.Context, artifact Artifact) (string, error) {
	client := m.Source.Client
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, artifact.URL, nil)
	if err != nil {
		return "", fmt.Errorf("create release download request: %w", err)
	}
	request.Header.Set("User-Agent", "lctl-updater")
	origin, err := url.Parse(artifact.URL)
	if err != nil || origin.Host == "" || origin.User != nil || (origin.Scheme != "https" && !m.Source.AllowHTTP) {
		return "", errors.New("release artifact has an invalid download URL")
	}
	response, err := trustedRedirectClient(client, origin, m.Source.AllowHTTP).Do(request)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", artifact.Filename, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", artifact.Filename, response.StatusCode)
	}
	if response.ContentLength > maxArchiveBytes {
		return "", fmt.Errorf("release archive is too large: %d bytes", response.ContentLength)
	}

	file, err := os.CreateTemp("", "lctl-update-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("create release archive staging file: %w", err)
	}
	path := file.Name()
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(path)
		}
	}()

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, maxArchiveBytes+1))
	if err != nil {
		return "", fmt.Errorf("download release archive: %w", err)
	}
	if written > maxArchiveBytes {
		return "", errors.New("release archive exceeds the size limit")
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); !strings.EqualFold(got, artifact.SHA256) {
		return "", fmt.Errorf("release archive checksum mismatch: got %s, want %s", got, artifact.SHA256)
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("sync release archive: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close release archive: %w", err)
	}
	remove = false
	return path, nil
}

func extractBinary(archivePath, destination string) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open release archive: %w", err)
	}
	defer archive.Close()
	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("open release gzip stream: %w", err)
	}
	defer gzipReader.Close()

	// Limit the decompressed stream as well as the download. Otherwise a small
	// gzip file with a huge non-binary entry could consume unbounded CPU while
	// tar.Reader skips to the next header.
	tarReader := tar.NewReader(io.LimitReader(gzipReader, maxExpandedArchiveBytes+1))
	found := false
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read release archive: %w", err)
		}

		cleanName := filepath.ToSlash(filepath.Clean(header.Name))
		if cleanName == "." || cleanName != header.Name || strings.Contains(header.Name, "\\") ||
			strings.HasPrefix(cleanName, "../") || strings.HasPrefix(cleanName, "/") {
			return fmt.Errorf("release archive contains unsafe path %q", header.Name)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return fmt.Errorf("release archive contains non-regular entry %q", header.Name)
		}
		if cleanName != "lctl" {
			continue
		}
		if found {
			return errors.New("release archive contains duplicate lctl binaries")
		}
		if header.Size < 1 || header.Size > maxBinaryBytes {
			return fmt.Errorf("release binary has invalid size %d", header.Size)
		}

		output, err := os.OpenFile(destination, os.O_WRONLY|os.O_TRUNC, 0700)
		if err != nil {
			return fmt.Errorf("open staged release binary: %w", err)
		}
		written, copyErr := io.Copy(output, io.LimitReader(tarReader, maxBinaryBytes+1))
		syncErr := output.Sync()
		closeErr := output.Close()
		if copyErr != nil {
			return fmt.Errorf("extract release binary: %w", copyErr)
		}
		if written != header.Size {
			return fmt.Errorf("release binary was truncated: got %d bytes, want %d", written, header.Size)
		}
		if syncErr != nil {
			return fmt.Errorf("sync staged release binary: %w", syncErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close staged release binary: %w", closeErr)
		}
		found = true
	}
	if !found {
		return errors.New("release archive does not contain lctl")
	}
	return nil
}

func validateStagedBinary(ctx context.Context, path, version string) error {
	validationContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	command := exec.CommandContext(validationContext, path, "version")
	command.Env = append(os.Environ(), "NO_COLOR=1", "TERM=dumb")
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("validate staged lctl binary: %w", err)
	}
	want := "lctl " + version
	if got := strings.TrimSpace(string(output)); got != want {
		return fmt.Errorf("staged lctl reported %q, want %q", got, want)
	}
	return nil
}

func replaceExecutable(target, staged string, mode os.FileMode) error {
	root := filepath.Dir(target)
	temporary, err := os.CreateTemp(root, ".lctl-update-*")
	if err != nil {
		return fmt.Errorf("create executable staging file beside %s: %w", target, err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	input, err := os.Open(staged)
	if err != nil {
		_ = temporary.Close()
		return fmt.Errorf("open staged lctl binary: %w", err)
	}
	_, copyErr := io.Copy(temporary, io.LimitReader(input, maxBinaryBytes+1))
	closeInputErr := input.Close()
	if copyErr != nil {
		_ = temporary.Close()
		return fmt.Errorf("stage replacement executable: %w", copyErr)
	}
	if closeInputErr != nil {
		_ = temporary.Close()
		return fmt.Errorf("close staged lctl binary: %w", closeInputErr)
	}
	if mode&0111 == 0 {
		mode |= 0755
	}
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set replacement executable permissions: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync replacement executable: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close replacement executable: %w", err)
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return fmt.Errorf("replace current executable: %w", err)
	}
	if err := syncDirectory(root); err != nil {
		return fmt.Errorf("sync executable directory: %w", err)
	}
	return nil
}

func isHomebrewPath(path string) bool {
	_, ok := homebrewLinkedExecutable(path)
	return ok
}

func homebrewLinkedExecutable(path string) (string, bool) {
	normalized := filepath.ToSlash(path)
	marker := "/Cellar/lctl/"
	markerIndex := strings.LastIndex(normalized, marker)
	if markerIndex <= 0 || !strings.HasSuffix(normalized, "/bin/lctl") {
		return "", false
	}
	return filepath.FromSlash(normalized[:markerIndex] + "/bin/lctl"), true
}

func runCommand(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdin = os.Stdin
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}
