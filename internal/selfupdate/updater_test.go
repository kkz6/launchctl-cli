package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestExtractBinaryAcceptsOnlyExpectedRegularFile(t *testing.T) {
	archive := makeTarGzip(t,
		tarTestEntry{name: "README.md", body: []byte("release notes")},
		tarTestEntry{name: "lctl", body: []byte("new lctl binary")},
	)
	archivePath := writeTestFile(t, "release.tar.gz", archive, 0600)
	destination := writeTestFile(t, "staged-lctl", []byte("old"), 0600)

	if err := extractBinary(archivePath, destination); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(contents), "new lctl binary"; got != want {
		t.Fatalf("extracted binary = %q, want %q", got, want)
	}
}

func TestExtractBinaryRejectsUnsafeArchives(t *testing.T) {
	tests := map[string][]tarTestEntry{
		"parent traversal": {{name: "../lctl", body: []byte("bad")}},
		"absolute path":    {{name: "/lctl", body: []byte("bad")}},
		"normalized path":  {{name: "payload/../lctl", body: []byte("bad")}},
		"dot path":         {{name: "./lctl", body: []byte("bad")}},
		"backslash path":   {{name: `payload\..\lctl`, body: []byte("bad")}},
		"nested binary":    {{name: "bin/lctl", body: []byte("bad")}},
		"symlink":          {{name: "lctl", typeflag: tar.TypeSymlink, linkname: "/tmp/evil"}},
		"directory":        {{name: "lctl", typeflag: tar.TypeDir}},
		"duplicate": {
			{name: "lctl", body: []byte("first")},
			{name: "lctl", body: []byte("second")},
		},
		"missing binary": {{name: "README.md", body: []byte("notes")}},
		"empty binary":   {{name: "lctl", body: nil}},
	}

	for name, entries := range tests {
		t.Run(name, func(t *testing.T) {
			archivePath := writeTestFile(t, "release.tar.gz", makeTarGzip(t, entries...), 0600)
			destination := writeTestFile(t, "staged-lctl", []byte("sentinel"), 0600)
			if err := extractBinary(archivePath, destination); err == nil {
				t.Fatal("extractBinary accepted an unsafe archive")
			}
		})
	}
}

func TestExtractBinaryRejectsCorruptGzip(t *testing.T) {
	archivePath := writeTestFile(t, "release.tar.gz", []byte("not gzip"), 0600)
	destination := writeTestFile(t, "staged-lctl", []byte("sentinel"), 0600)
	if err := extractBinary(archivePath, destination); err == nil || !strings.Contains(err.Error(), "gzip") {
		t.Fatalf("extractBinary error = %v, want gzip error", err)
	}
}

func TestDownloadArchiveVerifiesChecksum(t *testing.T) {
	body := makeTarGzip(t, tarTestEntry{name: "lctl", body: []byte("binary")})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(body)
	}))
	defer server.Close()

	manager := &Manager{Source: Source{Client: server.Client(), AllowHTTP: true}}
	artifact := Artifact{
		Filename: "lctl-linux-amd64.tar.gz",
		URL:      server.URL + "/v1.2.3/lctl-linux-amd64.tar.gz",
		SHA256:   strings.Repeat("0", 64),
	}
	path, err := manager.downloadArchive(context.Background(), artifact)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		if path != "" {
			_ = os.Remove(path)
		}
		t.Fatalf("downloadArchive error = %v, want checksum mismatch", err)
	}
}

func TestDownloadArchiveRejectsBadResponseAndDeclaredOversize(t *testing.T) {
	tests := map[string]struct {
		serve   func(http.ResponseWriter)
		wantErr string
	}{
		"HTTP status": {
			serve:   func(writer http.ResponseWriter) { writer.WriteHeader(http.StatusBadGateway) },
			wantErr: "HTTP 502",
		},
		"declared oversize": {
			serve: func(writer http.ResponseWriter) {
				writer.Header().Set("Content-Length", fmt.Sprint(maxArchiveBytes+1))
				writer.WriteHeader(http.StatusOK)
			},
			wantErr: "too large",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				test.serve(writer)
			}))
			defer server.Close()
			manager := &Manager{Source: Source{Client: server.Client(), AllowHTTP: true}}
			artifact := Artifact{Filename: "lctl-linux-amd64.tar.gz", URL: server.URL, SHA256: testDigest}
			if _, err := manager.downloadArchive(context.Background(), artifact); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("downloadArchive error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestManagerCheckUsesFreshCacheAndForcedRefresh(t *testing.T) {
	t.Run("fresh cache", func(t *testing.T) {
		now := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
		cache := Cache{Path: filepath.Join(t.TempDir(), "update.json"), Now: func() time.Time { return now }}
		if err := cache.RecordSuccess(testRelease("1.2.3", testDigest, "https://sin1.contabostorage.com/bucket/v1.2.3/lctl-linux-amd64.tar.gz")); err != nil {
			t.Fatal(err)
		}
		manager := &Manager{Cache: cache, Source: Source{GOOS: "unsupported", GOARCH: "unsupported"}}
		status, err := manager.Check(context.Background(), "1.0.0", false)
		if err != nil {
			t.Fatal(err)
		}
		if !status.UpdateAvailable || status.LatestVersion != "1.2.3" || status.CheckedAt == "" {
			t.Fatalf("unexpected cached status: %+v", status)
		}
	})

	t.Run("forced refresh", func(t *testing.T) {
		fixture := newUpdateFixture(t, "1.2.3", []byte("unused"))
		defer fixture.Close()
		manager := fixture.Manager(t, "/tmp/lctl")
		status, err := manager.Check(context.Background(), "1.0.0", true)
		if err != nil {
			t.Fatal(err)
		}
		if !status.UpdateAvailable || status.LatestVersion != "1.2.3" {
			t.Fatalf("unexpected refreshed status: %+v", status)
		}
		cached, found := manager.CachedStatus("1.0.0")
		if !found || cached.LatestVersion != "1.2.3" {
			t.Fatalf("refreshed release was not cached: found=%t status=%+v", found, cached)
		}
	})

	if _, err := (&Manager{}).Check(context.Background(), "dev", false); !errors.Is(err, ErrDevelopmentBuild) {
		t.Fatalf("Check(dev) error = %v, want ErrDevelopmentBuild", err)
	}
}

func TestExplicitOperationsDoNotDependOnWritableCache(t *testing.T) {
	for _, operation := range []string{"check", "update"} {
		t.Run(operation, func(t *testing.T) {
			fixture := newUpdateFixture(t, "1.2.3", []byte("unused"))
			defer fixture.Close()
			manager := fixture.Manager(t, "/tmp/lctl")
			regularFile := writeTestFile(t, "not-a-directory", []byte("blocked"), 0600)
			manager.Cache.Path = filepath.Join(regularFile, "update.json")

			if operation == "check" {
				status, err := manager.Check(context.Background(), "1.0.0", true)
				if err != nil || !status.UpdateAvailable {
					t.Fatalf("Check() status=%+v error=%v", status, err)
				}
				return
			}

			manager.Executable = func() (string, error) {
				t.Fatal("updater inspected the executable for an already-current release")
				return "", nil
			}
			result, err := manager.Update(context.Background(), "1.2.3", false, io.Discard, io.Discard)
			if err != nil || result.Updated {
				t.Fatalf("Update() result=%+v error=%v", result, err)
			}
		})
	}
}

func TestNewManagerConfiguresSafeDefaults(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	if manager.Source.Client == nil || manager.Cache.Path == "" || manager.Executable == nil || manager.EvalSymlinks == nil ||
		manager.LookPath == nil || manager.RunCommand == nil || manager.ValidateBinary == nil {
		t.Fatalf("manager has incomplete defaults: %+v", manager)
	}
}

func TestDirectUpdateReplacesExecutableAtomically(t *testing.T) {
	fixture := newUpdateFixture(t, "1.2.3", []byte("new executable"))
	defer fixture.Close()

	target := writeTestFile(t, "lctl", []byte("old executable"), 0750)
	if err := os.Chmod(target, 0750); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	validated := false
	manager := fixture.Manager(t, target)
	manager.ValidateBinary = func(_ context.Context, staged, version string) error {
		validated = true
		if version != "1.2.3" {
			return fmt.Errorf("version = %q", version)
		}
		contents, err := os.ReadFile(staged)
		if err != nil {
			return err
		}
		if string(contents) != "new executable" {
			return fmt.Errorf("staged contents = %q", contents)
		}
		return nil
	}

	result, err := manager.Update(context.Background(), "1.0.0", false, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !validated || !result.Updated || result.Method != "direct" || result.LatestVersion != "1.2.3" {
		t.Fatalf("unexpected result: validated=%t result=%+v", validated, result)
	}
	after, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Fatal("replacement reused the original executable inode instead of atomically renaming")
	}
	if got := after.Mode().Perm(); got != 0750 {
		t.Fatalf("replacement mode = %04o, want 0750", got)
	}
	contents, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "new executable" {
		t.Fatalf("replacement contents = %q", contents)
	}
	for _, pattern := range []string{".lctl-update-*", ".lctl-candidate-*"} {
		staging, err := filepath.Glob(filepath.Join(filepath.Dir(target), pattern))
		if err != nil {
			t.Fatal(err)
		}
		if len(staging) != 0 {
			t.Fatalf("replacement left staging files behind: %v", staging)
		}
	}
}

func TestUpdateSkipsInstallationWhenAlreadyCurrent(t *testing.T) {
	fixture := newUpdateFixture(t, "1.2.3", []byte("unused"))
	defer fixture.Close()
	manager := fixture.Manager(t, "/tmp/lctl")
	manager.Executable = func() (string, error) {
		t.Fatal("updater inspected the executable when no update was needed")
		return "", nil
	}

	result, err := manager.Update(context.Background(), "1.2.3", false, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated || result.UpdateAvailable {
		t.Fatalf("unexpected current-version result: %+v", result)
	}
}

func TestDirectUpdateLeavesExecutableUntouchedOnChecksumMismatch(t *testing.T) {
	fixture := newUpdateFixture(t, "1.2.3", []byte("new executable"))
	defer fixture.Close()
	fixture.digest = strings.Repeat("0", 64)

	target := writeTestFile(t, "lctl", []byte("old executable"), 0755)
	manager := fixture.Manager(t, target)
	manager.ValidateBinary = func(context.Context, string, string) error {
		t.Fatal("validator ran for an archive with a bad checksum")
		return nil
	}

	_, err := manager.Update(context.Background(), "1.0.0", false, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Update error = %v, want checksum mismatch", err)
	}
	assertFileContents(t, target, "old executable")
}

func TestDirectUpdateLeavesExecutableUntouchedOnVersionMismatch(t *testing.T) {
	fixture := newUpdateFixture(t, "1.2.3", []byte("new executable"))
	defer fixture.Close()

	target := writeTestFile(t, "lctl", []byte("old executable"), 0755)
	manager := fixture.Manager(t, target)
	manager.ValidateBinary = func(_ context.Context, _ string, version string) error {
		return fmt.Errorf("staged lctl reported %q, want %q", "1.2.2", version)
	}

	_, err := manager.Update(context.Background(), "1.0.0", false, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "staged lctl reported") {
		t.Fatalf("Update error = %v, want staged version mismatch", err)
	}
	assertFileContents(t, target, "old executable")
}

func TestUpdateRefusesForcedDowngrade(t *testing.T) {
	fixture := newUpdateFixture(t, "1.2.3", []byte("older executable"))
	defer fixture.Close()
	manager := fixture.Manager(t, "/tmp/lctl")
	manager.Executable = func() (string, error) {
		t.Fatal("updater inspected the executable during a refused downgrade")
		return "", nil
	}

	result, err := manager.Update(context.Background(), "2.0.0", true, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "refusing to downgrade") {
		t.Fatalf("Update error = %v, want downgrade refusal", err)
	}
	if result.Updated || result.CurrentVersion != "2.0.0" || result.LatestVersion != "1.2.3" {
		t.Fatalf("unexpected downgrade result: %+v", result)
	}
}

func TestHomebrewUpdateDelegatesToPackageManager(t *testing.T) {
	tests := map[string]struct {
		current  string
		force    bool
		wantArgs []string
	}{
		"upgrade":   {current: "1.0.0", wantArgs: []string{"upgrade", "kkz6/tap/lctl"}},
		"reinstall": {current: "1.2.3", force: true, wantArgs: []string{"reinstall", "kkz6/tap/lctl"}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newUpdateFixture(t, "1.2.3", []byte("unused"))
			defer fixture.Close()
			manager := fixture.Manager(t, "/opt/homebrew/bin/lctl")
			manager.EvalSymlinks = func(string) (string, error) {
				return "/opt/homebrew/Cellar/lctl/1.0.0/bin/lctl", nil
			}
			manager.LookPath = func(name string) (string, error) {
				if name != "brew" {
					t.Fatalf("LookPath(%q), want brew", name)
				}
				return "/opt/homebrew/bin/brew", nil
			}
			var gotName string
			var gotArgs []string
			var validatedPath, validatedVersion string
			manager.RunCommand = func(_ context.Context, command string, args []string, stdout, stderr io.Writer) error {
				gotName = command
				gotArgs = append([]string(nil), args...)
				_, _ = io.WriteString(stdout, "brew stdout")
				_, _ = io.WriteString(stderr, "brew stderr")
				return nil
			}
			manager.ValidateBinary = func(_ context.Context, path, version string) error {
				validatedPath, validatedVersion = path, version
				return nil
			}
			var stdout, stderr bytes.Buffer

			result, err := manager.Update(context.Background(), test.current, test.force, &stdout, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			if gotName != "/opt/homebrew/bin/brew" || !reflect.DeepEqual(gotArgs, test.wantArgs) {
				t.Fatalf("command = %q %v, want brew %v", gotName, gotArgs, test.wantArgs)
			}
			if !result.Updated || result.Method != "homebrew" {
				t.Fatalf("unexpected result: %+v", result)
			}
			if validatedPath != "/opt/homebrew/bin/lctl" || validatedVersion != "1.2.3" {
				t.Fatalf("validated %q as %q, want linked lctl 1.2.3", validatedPath, validatedVersion)
			}
			if stdout.String() != "brew stdout" || stderr.String() != "brew stderr" {
				t.Fatalf("command writers not forwarded: stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestHomebrewUpdateRejectsSuccessfulNoOp(t *testing.T) {
	fixture := newUpdateFixture(t, "1.2.3", []byte("unused"))
	defer fixture.Close()
	manager := fixture.Manager(t, "/opt/homebrew/bin/lctl")
	manager.EvalSymlinks = func(string) (string, error) {
		return "/opt/homebrew/Cellar/lctl/1.0.0/bin/lctl", nil
	}
	manager.LookPath = func(string) (string, error) { return "/opt/homebrew/bin/brew", nil }
	manager.RunCommand = func(context.Context, string, []string, io.Writer, io.Writer) error { return nil }
	manager.ValidateBinary = func(_ context.Context, path, version string) error {
		if path != "/opt/homebrew/bin/lctl" || version != "1.2.3" {
			t.Fatalf("ValidateBinary(%q, %q)", path, version)
		}
		return errors.New(`staged lctl reported "lctl 1.0.0", want "lctl 1.2.3"`)
	}

	result, err := manager.Update(context.Background(), "1.0.0", false, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "installed lctl validation failed") {
		t.Fatalf("Update error = %v, want post-Homebrew validation failure", err)
	}
	if result.Updated {
		t.Fatalf("failed Homebrew validation reported success: %+v", result)
	}
}

func TestHomebrewUpdateExplainsMissingBrewAndTrustFailure(t *testing.T) {
	fixture := newUpdateFixture(t, "1.2.3", []byte("unused"))
	defer fixture.Close()
	manager := fixture.Manager(t, "/opt/homebrew/bin/lctl")
	manager.EvalSymlinks = func(string) (string, error) {
		return "/opt/homebrew/Cellar/lctl/1.0.0/bin/lctl", nil
	}

	t.Run("missing brew", func(t *testing.T) {
		manager.LookPath = func(string) (string, error) { return "", errors.New("missing") }
		_, err := manager.Update(context.Background(), "1.0.0", false, io.Discard, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "brew is not available") {
			t.Fatalf("Update error = %v", err)
		}
	})

	t.Run("command failure", func(t *testing.T) {
		manager.LookPath = func(string) (string, error) { return "/opt/homebrew/bin/brew", nil }
		manager.RunCommand = func(context.Context, string, []string, io.Writer, io.Writer) error {
			return errors.New("formula is not trusted")
		}
		_, err := manager.Update(context.Background(), "1.0.0", false, io.Discard, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "brew trust --formula kkz6/tap/lctl") {
			t.Fatalf("Update error = %v, want trust guidance", err)
		}
	})
}

func TestUpdateRefusesNonHomebrewSymlink(t *testing.T) {
	fixture := newUpdateFixture(t, "1.2.3", []byte("unused"))
	defer fixture.Close()
	manager := fixture.Manager(t, "/usr/local/bin/lctl")
	manager.EvalSymlinks = func(string) (string, error) { return "/srv/tools/lctl", nil }

	_, err := manager.Update(context.Background(), "1.0.0", false, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "refusing to replace symlinked executable") {
		t.Fatalf("Update error = %v, want symlink refusal", err)
	}
}

func TestValidateStagedBinaryRequiresExactVersion(t *testing.T) {
	tests := map[string]struct {
		output  string
		wantErr bool
	}{
		"match":    {output: "lctl 1.2.3\n"},
		"mismatch": {output: "lctl 1.2.2\n", wantErr: true},
		"noise":    {output: "notice\nlctl 1.2.3\n", wantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			path := writeTestFile(t, "candidate", []byte("#!/bin/sh\nprintf '%s' '"+test.output+"'\n"), 0700)
			if err := os.Chmod(path, 0700); err != nil {
				t.Fatal(err)
			}
			err := validateStagedBinary(context.Background(), path, "1.2.3")
			if (err != nil) != test.wantErr {
				t.Fatalf("validateStagedBinary error = %v, wantErr=%t", err, test.wantErr)
			}
		})
	}
}

func TestHomebrewPathDetectionIsSpecific(t *testing.T) {
	tests := map[string]bool{
		"/opt/homebrew/Cellar/lctl/1.2.3/bin/lctl":  true,
		"/usr/local/Cellar/lctl/1.2.3/bin/lctl":     true,
		"/opt/homebrew/bin/lctl":                    false,
		"/opt/homebrew/Cellar/other/1.2.3/bin/lctl": false,
		"/tmp/Cellar/lctl/1.2.3/bin/not-lctl":       false,
	}
	for path, want := range tests {
		if got := isHomebrewPath(path); got != want {
			t.Errorf("isHomebrewPath(%q) = %t, want %t", path, got, want)
		}
	}
}

func TestHomebrewLinkedExecutable(t *testing.T) {
	tests := map[string]string{
		"/opt/homebrew/Cellar/lctl/1.2.3/bin/lctl":              "/opt/homebrew/bin/lctl",
		"/usr/local/Cellar/lctl/1.2.3_1/bin/lctl":               "/usr/local/bin/lctl",
		"/home/linuxbrew/.linuxbrew/Cellar/lctl/1.2.3/bin/lctl": "/home/linuxbrew/.linuxbrew/bin/lctl",
	}
	for resolved, want := range tests {
		got, ok := homebrewLinkedExecutable(resolved)
		if !ok || got != want {
			t.Errorf("homebrewLinkedExecutable(%q) = %q, %t; want %q, true", resolved, got, ok, want)
		}
	}
	if got, ok := homebrewLinkedExecutable("/usr/local/bin/lctl"); ok || got != "" {
		t.Fatalf("direct binary resolved as Homebrew: %q, %t", got, ok)
	}
}

type tarTestEntry struct {
	name     string
	body     []byte
	typeflag byte
	linkname string
}

func makeTarGzip(t *testing.T, entries ...tarTestEntry) []byte {
	t.Helper()
	var compressed bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressed)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{
			Name:     entry.name,
			Mode:     0755,
			Size:     int64(len(entry.body)),
			Typeflag: typeflag,
			Linkname: entry.linkname,
		}
		if typeflag != tar.TypeReg && typeflag != tar.TypeRegA {
			header.Size = 0
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if len(entry.body) > 0 {
			if _, err := tarWriter.Write(entry.body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return compressed.Bytes()
}

func writeTestFile(t *testing.T, name string, contents []byte, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, contents, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(contents); got != want {
		t.Fatalf("%s contents = %q, want %q", path, got, want)
	}
}

type updateFixture struct {
	server  *httptest.Server
	version string
	archive []byte
	digest  string
}

func newUpdateFixture(t *testing.T, version string, binary []byte) *updateFixture {
	t.Helper()
	fixture := &updateFixture{
		version: version,
		archive: makeTarGzip(t, tarTestEntry{name: "lctl", body: binary}),
	}
	digest := sha256.Sum256(fixture.archive)
	fixture.digest = fmt.Sprintf("%x", digest[:])
	fixture.server = httptest.NewServer(http.HandlerFunc(fixture.serveHTTP))
	return fixture
}

func (f *updateFixture) Close() {
	f.server.Close()
}

func (f *updateFixture) artifactURL() string {
	return f.server.URL + "/v" + f.version + "/lctl-linux-amd64.tar.gz"
}

func (f *updateFixture) Source() Source {
	return Source{
		Client:      f.server.Client(),
		ManifestURL: f.server.URL + "/latest.json",
		FormulaURL:  f.server.URL + "/formula.rb",
		GOOS:        "linux",
		GOARCH:      "amd64",
		AllowHTTP:   true,
	}
}

func (f *updateFixture) Manager(t *testing.T, executable string) *Manager {
	t.Helper()
	return &Manager{
		Source:     f.Source(),
		Cache:      Cache{Path: filepath.Join(t.TempDir(), "update.json"), Now: func() time.Time { return time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC) }},
		Executable: func() (string, error) { return executable, nil },
		EvalSymlinks: func(path string) (string, error) {
			return path, nil
		},
		LookPath: func(name string) (string, error) { return name, nil },
		RunCommand: func(context.Context, string, []string, io.Writer, io.Writer) error {
			return nil
		},
		ValidateBinary: func(context.Context, string, string) error { return nil },
	}
}

func (f *updateFixture) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/latest.json":
		fmt.Fprintf(writer, `{"schema_version":1,"channel":"stable","version":%q,"tag":%q,"artifacts":[{"os":"linux","arch":"amd64","filename":"lctl-linux-amd64.tar.gz","url":%q,"sha256":%q}]}`,
			f.version, "v"+f.version, f.artifactURL(), f.digest)
	case "/formula.rb":
		fmt.Fprintf(writer, "version %q\nurl %q\nsha256 %q\n", f.version, f.artifactURL(), f.digest)
	case "/v" + f.version + "/lctl-linux-amd64.tar.gz":
		writer.Header().Set("Content-Type", "application/gzip")
		_, _ = writer.Write(f.archive)
	default:
		http.NotFound(writer, request)
	}
}
