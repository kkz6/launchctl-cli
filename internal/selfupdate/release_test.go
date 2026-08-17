package selfupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

const testDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestDefaultSourceUsesRuntimePlatformAndProvidedClient(t *testing.T) {
	client := &http.Client{}
	source := DefaultSource(client)
	if source.Client != client || source.GOOS != runtime.GOOS || source.GOARCH != runtime.GOARCH {
		t.Fatalf("unexpected default source: %+v", source)
	}
	if source.ManifestURL != DefaultManifestURL || source.FormulaURL != DefaultFormulaURL || source.AllowHTTP {
		t.Fatalf("unexpected default release endpoints: %+v", source)
	}
}

func TestSourceFetchRequiresMatchingManifestAndFormula(t *testing.T) {
	fixture := newMetadataFixture(t, "1.2.3", testDigest)
	defer fixture.Close()

	release, err := fixture.Source().Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if release.Version != "1.2.3" || release.Source != "manifest+formula" {
		t.Fatalf("unexpected release: %+v", release)
	}
	if release.Artifact.Filename != "lctl-linux-amd64.tar.gz" || release.Artifact.SHA256 != testDigest {
		t.Fatalf("unexpected artifact: %+v", release.Artifact)
	}
}

func TestSourceFetchFallsBackWhenOneMetadataSourceIsUnavailable(t *testing.T) {
	tests := map[string]struct {
		manifestStatus int
		formulaStatus  int
		wantSource     string
	}{
		"formula fallback":  {manifestStatus: http.StatusNotFound, formulaStatus: http.StatusOK, wantSource: "homebrew-formula"},
		"manifest fallback": {manifestStatus: http.StatusOK, formulaStatus: http.StatusNotFound, wantSource: "manifest"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newMetadataFixture(t, "1.2.3", testDigest)
			defer fixture.Close()
			fixture.manifestStatus = test.manifestStatus
			fixture.formulaStatus = test.formulaStatus

			release, err := fixture.Source().Fetch(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if release.Source != test.wantSource {
				t.Fatalf("source = %q, want %q", release.Source, test.wantSource)
			}
		})
	}
}

func TestSourceFetchRejectsMetadataDisagreement(t *testing.T) {
	tests := map[string]func(*metadataFixture){
		"version": func(f *metadataFixture) { f.formulaVersion = "1.2.4" },
		"URL": func(f *metadataFixture) {
			f.formulaArtifactURL = f.server.URL + "/mirror/v1.2.3/lctl-linux-amd64.tar.gz"
		},
		"digest": func(f *metadataFixture) { f.formulaDigest = strings.Repeat("a", 64) },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newMetadataFixture(t, "1.2.3", testDigest)
			defer fixture.Close()
			mutate(fixture)

			_, err := fixture.Source().Fetch(context.Background())
			if err == nil || !strings.Contains(err.Error(), "does not match") {
				t.Fatalf("Fetch() error = %v, want metadata mismatch", err)
			}
		})
	}
}

func TestSourceFetchDoesNotTrustMalformedFallbackMetadata(t *testing.T) {
	t.Run("malformed manifest", func(t *testing.T) {
		fixture := newMetadataFixture(t, "1.2.3", testDigest)
		defer fixture.Close()
		fixture.mutateManifest = func(manifest *Manifest) { manifest.Tag = "v9.9.9" }

		_, err := fixture.Source().Fetch(context.Background())
		if err == nil || !strings.Contains(err.Error(), "does not match the Homebrew formula") {
			t.Fatalf("Fetch error = %v, want trusted-source mismatch", err)
		}
	})

	t.Run("malformed formula", func(t *testing.T) {
		fixture := newMetadataFixture(t, "1.2.3", testDigest)
		defer fixture.Close()
		fixture.formulaDigest = "not-a-digest"

		_, err := fixture.Source().Fetch(context.Background())
		if err == nil || !strings.Contains(err.Error(), "does not match the Homebrew formula") {
			t.Fatalf("Fetch error = %v, want trusted-source mismatch", err)
		}
	})
}

func TestManifestValidation(t *testing.T) {
	tests := map[string]struct {
		mutate  func(*Manifest)
		wantErr string
	}{
		"unsupported schema": {
			mutate:  func(m *Manifest) { m.SchemaVersion = 2 },
			wantErr: "unsupported update manifest",
		},
		"non-stable channel": {
			mutate:  func(m *Manifest) { m.Channel = "beta" },
			wantErr: "unsupported update manifest",
		},
		"invalid version": {
			mutate:  func(m *Manifest) { m.Version = "latest" },
			wantErr: "invalid release version",
		},
		"prerelease on stable channel": {
			mutate:  func(m *Manifest) { m.Version = "1.2.3-rc.1" },
			wantErr: "stable x.y.z releases only",
		},
		"tag mismatch": {
			mutate:  func(m *Manifest) { m.Tag = "v1.2.4" },
			wantErr: "does not match version",
		},
		"missing platform": {
			mutate:  func(m *Manifest) { m.Artifacts[0].Arch = "arm64" },
			wantErr: "0 artifacts for linux/amd64",
		},
		"duplicate platform": {
			mutate:  func(m *Manifest) { m.Artifacts = append(m.Artifacts, m.Artifacts[0]) },
			wantErr: "2 artifacts for linux/amd64",
		},
		"filename mismatch": {
			mutate:  func(m *Manifest) { m.Artifacts[0].Filename = "lctl-linux-arm64.tar.gz" },
			wantErr: "does not match linux/amd64",
		},
		"invalid digest": {
			mutate:  func(m *Manifest) { m.Artifacts[0].SHA256 = "not-a-digest" },
			wantErr: "invalid SHA-256",
		},
		"wrong artifact version": {
			mutate: func(m *Manifest) {
				m.Artifacts[0].URL = "http://example.test/v9.9.9/lctl-linux-amd64.tar.gz"
			},
			wantErr: "does not contain version",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newMetadataFixture(t, "1.2.3", testDigest)
			defer fixture.Close()
			fixture.formulaStatus = http.StatusNotFound
			fixture.mutateManifest = test.mutate

			_, err := fixture.Source().Fetch(context.Background())
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Fetch() error = %v, want error containing %q", err, test.wantErr)
			}
		})
	}
}

func TestFormulaParserSelectsRequestedPlatform(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/formula.rb" {
			http.NotFound(writer, request)
			return
		}
		fmt.Fprintf(writer, `class Lctl < Formula
  version "2.4.6"
  on_macos do
    url "%s/v2.4.6/lctl-darwin-arm64.tar.gz"
    sha256 "%s"
  end
  on_linux do
    url "%s/v2.4.6/lctl-linux-amd64.tar.gz"
    sha256 "%s"
  end
end
`, server.URL, strings.Repeat("a", 64), server.URL, testDigest)
	}))
	defer server.Close()

	source := Source{
		Client:      server.Client(),
		FormulaURL:  server.URL + "/formula.rb",
		GOOS:        "linux",
		GOARCH:      "amd64",
		AllowHTTP:   true,
		ManifestURL: "",
	}
	release, err := source.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if release.Version != "2.4.6" || release.Artifact.SHA256 != testDigest {
		t.Fatalf("unexpected release: %+v", release)
	}
}

func TestFormulaParserRejectsAmbiguousMetadata(t *testing.T) {
	tests := map[string]string{
		"duplicate version": `version "1.2.3"
version "1.2.3"
url "http://example.test/v1.2.3/lctl-linux-amd64.tar.gz"
sha256 "` + testDigest + `"`,
		"duplicate artifact": `version "1.2.3"
url "http://example.test/v1.2.3/lctl-linux-amd64.tar.gz"
sha256 "` + testDigest + `"
url "http://example.test/v1.2.3/lctl-linux-amd64.tar.gz"
sha256 "` + testDigest + `"`,
	}

	for name, formula := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(formula))
			}))
			defer server.Close()

			source := Source{Client: server.Client(), FormulaURL: server.URL, GOOS: "linux", GOARCH: "amd64", AllowHTTP: true}
			_, err := source.Fetch(context.Background())
			if err == nil {
				t.Fatal("Fetch() succeeded with ambiguous formula")
			}
		})
	}
}

func TestMetadataFetchGuardsTransportAndSize(t *testing.T) {
	t.Run("HTTPS required", func(t *testing.T) {
		source := Source{Client: http.DefaultClient}
		if _, err := source.fetch(context.Background(), "http://example.test/latest.json"); err == nil {
			t.Fatal("fetch accepted an insecure metadata URL")
		}
	})

	t.Run("status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusBadGateway)
		}))
		defer server.Close()
		source := Source{Client: server.Client(), AllowHTTP: true}
		if _, err := source.fetch(context.Background(), server.URL); err == nil || !strings.Contains(err.Error(), "HTTP 502") {
			t.Fatalf("fetch error = %v", err)
		}
	})

	t.Run("streaming limit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.(http.Flusher).Flush()
			_, _ = writer.Write([]byte(strings.Repeat("x", maxMetadataBytes+1)))
		}))
		defer server.Close()
		source := Source{Client: server.Client(), AllowHTTP: true}
		if _, err := source.fetch(context.Background(), server.URL); err == nil || !strings.Contains(err.Error(), "size limit") {
			t.Fatalf("fetch error = %v", err)
		}
	})
}

func TestMetadataFetchRejectsUntrustedRedirects(t *testing.T) {
	t.Run("cross-host", func(t *testing.T) {
		var targetRequests int
		target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			targetRequests++
			_, _ = writer.Write([]byte("should not be reached"))
		}))
		defer target.Close()
		origin := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			http.Redirect(writer, &http.Request{}, target.URL+"/latest.json", http.StatusFound)
		}))
		defer origin.Close()

		source := Source{Client: origin.Client(), AllowHTTP: true}
		_, err := source.fetch(context.Background(), origin.URL)
		if err == nil || !errors.Is(err, errUntrustedRedirect) {
			t.Fatalf("fetch error = %v, want errUntrustedRedirect", err)
		}
		if targetRequests != 0 {
			t.Fatalf("redirect target received %d requests, want 0", targetRequests)
		}
	})

	t.Run("HTTPS downgrade", func(t *testing.T) {
		var server *httptest.Server
		server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			location := "http://" + request.Host + "/latest.json"
			http.Redirect(writer, request, location, http.StatusFound)
		}))
		defer server.Close()

		source := Source{Client: server.Client()}
		_, err := source.fetch(context.Background(), server.URL)
		if err == nil || !errors.Is(err, errUntrustedRedirect) {
			t.Fatalf("fetch error = %v, want HTTPS downgrade rejection", err)
		}
	})

	t.Run("same-host HTTPS", func(t *testing.T) {
		var server *httptest.Server
		server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path == "/start" {
				http.Redirect(writer, request, "/final", http.StatusFound)
				return
			}
			_, _ = writer.Write([]byte("metadata"))
		}))
		defer server.Close()

		source := Source{Client: server.Client()}
		data, err := source.fetch(context.Background(), server.URL+"/start")
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "metadata" {
			t.Fatalf("redirect response = %q", data)
		}
	})
}

func TestReleaseValidationRestrictsPlatformAndArtifactHost(t *testing.T) {
	t.Run("unsupported platform", func(t *testing.T) {
		source := Source{GOOS: "windows", GOARCH: "amd64"}
		if _, err := source.Fetch(context.Background()); err == nil || !strings.Contains(err.Error(), "unsupported update platform") {
			t.Fatalf("Fetch error = %v", err)
		}
	})

	t.Run("untrusted artifact host", func(t *testing.T) {
		source := Source{GOOS: "linux", GOARCH: "amd64"}
		release := testRelease("1.2.3", testDigest, "https://evil.example/v1.2.3/lctl-linux-amd64.tar.gz")
		if _, err := source.validateRelease(release); err == nil || !strings.Contains(err.Error(), "not trusted") {
			t.Fatalf("validateRelease error = %v", err)
		}
	})

	t.Run("untrusted bucket on shared host", func(t *testing.T) {
		source := Source{GOOS: "linux", GOARCH: "amd64"}
		release := testRelease("1.2.3", testDigest,
			"https://sin1.contabostorage.com/attacker-bucket/v1.2.3/lctl-linux-amd64.tar.gz")
		if _, err := source.validateRelease(release); err == nil || !strings.Contains(err.Error(), "not trusted") {
			t.Fatalf("validateRelease error = %v", err)
		}
	})

	t.Run("path traversal from official bucket", func(t *testing.T) {
		source := Source{GOOS: "linux", GOARCH: "amd64"}
		release := testRelease("1.2.3", testDigest,
			"https://sin1.contabostorage.com/2fac7399ecc245c4b352abf9eb154e1d:launchctl-cli/v1.2.3/../../attacker-bucket/v1.2.3/lctl-linux-amd64.tar.gz")
		if _, err := source.validateRelease(release); err == nil || !strings.Contains(err.Error(), "not trusted") {
			t.Fatalf("validateRelease error = %v", err)
		}
	})

	t.Run("official bucket", func(t *testing.T) {
		source := Source{GOOS: "linux", GOARCH: "amd64"}
		release := testRelease("1.2.3", testDigest,
			"https://sin1.contabostorage.com/2fac7399ecc245c4b352abf9eb154e1d:launchctl-cli/v1.2.3/lctl-linux-amd64.tar.gz")
		if _, err := source.validateRelease(release); err != nil {
			t.Fatalf("validateRelease rejected official artifact URL: %v", err)
		}
	})
}

func TestStatusForSemanticVersions(t *testing.T) {
	tests := map[string]struct {
		current   string
		latest    string
		available bool
	}{
		"newer release":       {current: "1.2.3", latest: "1.2.4", available: true},
		"same release":        {current: "v1.2.3", latest: "1.2.3", available: false},
		"never downgrade":     {current: "2.0.0", latest: "1.9.9", available: false},
		"prerelease to final": {current: "1.2.3-rc.1", latest: "1.2.3", available: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			status, err := StatusFor(test.current, Release{Version: test.latest})
			if err != nil {
				t.Fatal(err)
			}
			if status.UpdateAvailable != test.available {
				t.Fatalf("UpdateAvailable = %t, want %t", status.UpdateAvailable, test.available)
			}
		})
	}

	if _, err := StatusFor("dev", Release{Version: "1.2.3"}); !errors.Is(err, ErrDevelopmentBuild) {
		t.Fatalf("StatusFor(dev) error = %v, want ErrDevelopmentBuild", err)
	}
}

type metadataFixture struct {
	server             *httptest.Server
	version            string
	digest             string
	manifestStatus     int
	formulaStatus      int
	formulaVersion     string
	formulaDigest      string
	formulaArtifactURL string
	mutateManifest     func(*Manifest)
}

func newMetadataFixture(t *testing.T, version, digest string) *metadataFixture {
	t.Helper()
	fixture := &metadataFixture{
		version:        version,
		digest:         digest,
		manifestStatus: http.StatusOK,
		formulaStatus:  http.StatusOK,
		formulaVersion: version,
		formulaDigest:  digest,
	}
	fixture.server = httptest.NewServer(http.HandlerFunc(fixture.serveHTTP))
	fixture.formulaArtifactURL = fixture.artifactURL(version)
	return fixture
}

func (f *metadataFixture) Close() {
	f.server.Close()
}

func (f *metadataFixture) Source() Source {
	return Source{
		Client:      f.server.Client(),
		ManifestURL: f.server.URL + "/latest.json",
		FormulaURL:  f.server.URL + "/formula.rb",
		GOOS:        "linux",
		GOARCH:      "amd64",
		AllowHTTP:   true,
	}
}

func (f *metadataFixture) artifactURL(version string) string {
	return f.server.URL + "/v" + version + "/lctl-linux-amd64.tar.gz"
}

func (f *metadataFixture) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/latest.json":
		if f.manifestStatus != http.StatusOK {
			writer.WriteHeader(f.manifestStatus)
			return
		}
		manifest := Manifest{
			SchemaVersion: 1,
			Channel:       "stable",
			Version:       f.version,
			Tag:           "v" + f.version,
			PublishedAt:   "2026-08-17T00:00:00Z",
			Artifacts: []Artifact{{
				OS:       "linux",
				Arch:     "amd64",
				Filename: "lctl-linux-amd64.tar.gz",
				URL:      f.artifactURL(f.version),
				SHA256:   f.digest,
			}},
		}
		if f.mutateManifest != nil {
			f.mutateManifest(&manifest)
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(manifest)
	case "/formula.rb":
		if f.formulaStatus != http.StatusOK {
			writer.WriteHeader(f.formulaStatus)
			return
		}
		fmt.Fprintf(writer, "class Lctl < Formula\n  version %q\n  url %q\n  sha256 %q\nend\n", f.formulaVersion, f.formulaArtifactURL, f.formulaDigest)
	default:
		http.NotFound(writer, request)
	}
}
