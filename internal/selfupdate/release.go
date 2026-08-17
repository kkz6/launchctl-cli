// Package selfupdate checks for and installs lctl releases.
package selfupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

const (
	DefaultManifestURL  = "https://sin1.contabostorage.com/2fac7399ecc245c4b352abf9eb154e1d:launchctl-cli/latest.json"
	DefaultFormulaURL   = "https://raw.githubusercontent.com/kkz6/homebrew-tap/main/Formula/lctl.rb"
	maxMetadataBytes    = 1 << 20
	trustedArtifactHost = "sin1.contabostorage.com"
	trustedArtifactRoot = "/2fac7399ecc245c4b352abf9eb154e1d:launchctl-cli/"
)

var (
	ErrDevelopmentBuild    = errors.New("development builds cannot be updated automatically")
	errMetadataUnavailable = errors.New("update metadata is unavailable")
	errUntrustedRedirect   = errors.New("untrusted update redirect")
	versionPattern         = regexp.MustCompile(`(?m)^\s*version\s+"([^"]+)"\s*$`)
	stableVersionPattern   = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+$`)
	sha256Pattern          = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

type Artifact struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
}

type Manifest struct {
	SchemaVersion int        `json:"schema_version"`
	Channel       string     `json:"channel"`
	Version       string     `json:"version"`
	Tag           string     `json:"tag"`
	PublishedAt   string     `json:"published_at,omitempty"`
	Artifacts     []Artifact `json:"artifacts"`
}

type Release struct {
	Version     string   `json:"version"`
	Artifact    Artifact `json:"-"`
	PublishedAt string   `json:"published_at,omitempty"`
	Source      string   `json:"source,omitempty"`
}

type Status struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	CheckedAt       string `json:"checked_at,omitempty"`
	Source          string `json:"source,omitempty"`
}

type Source struct {
	Client      *http.Client
	ManifestURL string
	FormulaURL  string
	GOOS        string
	GOARCH      string
	AllowHTTP   bool
}

func DefaultSource(client *http.Client) Source {
	return Source{
		Client:      client,
		ManifestURL: DefaultManifestURL,
		FormulaURL:  DefaultFormulaURL,
		GOOS:        runtime.GOOS,
		GOARCH:      runtime.GOARCH,
	}
}

func (s Source) Fetch(ctx context.Context) (Release, error) {
	if s.Client == nil {
		s.Client = http.DefaultClient
	}
	if s.GOOS == "" {
		s.GOOS = runtime.GOOS
	}
	if s.GOARCH == "" {
		s.GOARCH = runtime.GOARCH
	}
	if !supportedPlatform(s.GOOS, s.GOARCH) {
		return Release{}, fmt.Errorf("unsupported update platform %s/%s", s.GOOS, s.GOARCH)
	}

	type metadataResult struct {
		kind    string
		release Release
		err     error
	}
	results := make(chan metadataResult, 2)
	go func() {
		release, err := s.fetchManifest(ctx)
		results <- metadataResult{kind: "manifest", release: release, err: err}
	}()
	go func() {
		release, err := s.fetchFormula(ctx)
		results <- metadataResult{kind: "formula", release: release, err: err}
	}()

	var manifestRelease, formulaRelease Release
	var manifestErr, formulaErr error
	for range 2 {
		result := <-results
		if result.kind == "manifest" {
			manifestRelease, manifestErr = result.release, result.err
		} else {
			formulaRelease, formulaErr = result.release, result.err
		}
	}

	switch {
	case manifestErr == nil && formulaErr == nil:
		if manifestRelease.Version != formulaRelease.Version ||
			manifestRelease.Artifact.URL != formulaRelease.Artifact.URL ||
			!strings.EqualFold(manifestRelease.Artifact.SHA256, formulaRelease.Artifact.SHA256) {
			return Release{}, errors.New("release manifest does not match the Homebrew formula")
		}
		manifestRelease.Source = "manifest+formula"
		return manifestRelease, nil
	case formulaErr == nil && errors.Is(manifestErr, errMetadataUnavailable):
		return formulaRelease, nil
	case manifestErr == nil && errors.Is(formulaErr, errMetadataUnavailable):
		return manifestRelease, nil
	case formulaErr == nil:
		return Release{}, fmt.Errorf("release manifest does not match the Homebrew formula: %v", manifestErr)
	case manifestErr == nil:
		return Release{}, fmt.Errorf("release manifest does not match the Homebrew formula: %v", formulaErr)
	default:
		return Release{}, fmt.Errorf("discover latest release: manifest: %v; formula: %v", manifestErr, formulaErr)
	}
}

func (s Source) fetchManifest(ctx context.Context) (Release, error) {
	if s.ManifestURL == "" {
		return Release{}, fmt.Errorf("%w: manifest source is disabled", errMetadataUnavailable)
	}
	data, err := s.fetch(ctx, s.ManifestURL)
	if err != nil {
		return Release{}, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Release{}, fmt.Errorf("decode update manifest: %w", err)
	}
	if manifest.SchemaVersion != 1 || manifest.Channel != "stable" {
		return Release{}, fmt.Errorf("unsupported update manifest schema %d channel %q", manifest.SchemaVersion, manifest.Channel)
	}

	var matches []Artifact
	for _, artifact := range manifest.Artifacts {
		if artifact.OS == s.GOOS && artifact.Arch == s.GOARCH {
			matches = append(matches, artifact)
		}
	}
	if len(matches) != 1 {
		return Release{}, fmt.Errorf("update manifest has %d artifacts for %s/%s", len(matches), s.GOOS, s.GOARCH)
	}
	release := Release{
		Version:     manifest.Version,
		Artifact:    matches[0],
		PublishedAt: manifest.PublishedAt,
		Source:      "manifest",
	}
	release, err = s.validateRelease(release)
	if err != nil {
		return Release{}, err
	}
	if manifest.Tag != "v"+release.Version {
		return Release{}, fmt.Errorf("update manifest tag %q does not match version %q", manifest.Tag, release.Version)
	}
	return release, nil
}

func (s Source) fetchFormula(ctx context.Context) (Release, error) {
	if s.FormulaURL == "" {
		return Release{}, fmt.Errorf("%w: formula source is disabled", errMetadataUnavailable)
	}
	data, err := s.fetch(ctx, s.FormulaURL)
	if err != nil {
		return Release{}, err
	}

	versionMatches := versionPattern.FindAllSubmatch(data, -1)
	if len(versionMatches) != 1 {
		return Release{}, fmt.Errorf("Homebrew formula contains %d release versions", len(versionMatches))
	}

	filename := artifactFilename(s.GOOS, s.GOARCH)
	assetPattern := regexp.MustCompile(`(?m)^\s*url\s+"([^"]*/` + regexp.QuoteMeta(filename) + `)"\s*\n\s*sha256\s+"([a-fA-F0-9]{64})"\s*$`)
	assetMatches := assetPattern.FindAllSubmatch(data, -1)
	if len(assetMatches) != 1 {
		return Release{}, fmt.Errorf("Homebrew formula has %d matching artifacts for %s", len(assetMatches), filename)
	}

	release := Release{
		Version: string(versionMatches[0][1]),
		Artifact: Artifact{
			OS:       s.GOOS,
			Arch:     s.GOARCH,
			Filename: filename,
			URL:      string(assetMatches[0][1]),
			SHA256:   strings.ToLower(string(assetMatches[0][2])),
		},
		Source: "homebrew-formula",
	}
	return s.validateRelease(release)
}

func (s Source) fetch(ctx context.Context, location string) ([]byte, error) {
	parsed, err := url.Parse(location)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && !s.AllowHTTP) {
		return nil, fmt.Errorf("invalid update metadata URL %q", location)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, fmt.Errorf("create update metadata request: %w", err)
	}
	request.Header.Set("Accept", "application/json, text/plain;q=0.9")
	request.Header.Set("User-Agent", "lctl-updater")

	response, err := trustedRedirectClient(s.Client, parsed, s.AllowHTTP).Do(request)
	if err != nil {
		if errors.Is(err, errUntrustedRedirect) {
			return nil, fmt.Errorf("fetch %s: %w", parsed.Host, err)
		}
		return nil, fmt.Errorf("%w: fetch %s: %v", errMetadataUnavailable, parsed.Host, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: fetch %s: HTTP %d", errMetadataUnavailable, parsed.Host, response.StatusCode)
	}
	if response.ContentLength > maxMetadataBytes {
		return nil, fmt.Errorf("update metadata is too large: %d bytes", response.ContentLength)
	}

	limited := io.LimitReader(response.Body, maxMetadataBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read update metadata: %w", err)
	}
	if len(data) > maxMetadataBytes {
		return nil, errors.New("update metadata exceeds the size limit")
	}
	return data, nil
}

// trustedRedirectClient keeps update requests on the origin selected by the
// release contract. The updater never needs to follow a cross-host redirect,
// and doing so would let compromised metadata turn an update check into an
// arbitrary network request.
func trustedRedirectClient(client *http.Client, origin *url.URL, allowHTTP bool) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	clone := *client
	previous := client.CheckRedirect
	clone.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if request.URL == nil || !strings.EqualFold(request.URL.Host, origin.Host) {
			return fmt.Errorf("%w: redirect changed the trusted host", errUntrustedRedirect)
		}
		if request.URL.Scheme != "https" && !allowHTTP {
			return fmt.Errorf("%w: redirect used an insecure scheme", errUntrustedRedirect)
		}
		if previous != nil {
			return previous(request, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 update redirects")
		}
		return nil
	}
	return &clone
}

func (s Source) validateRelease(release Release) (Release, error) {
	if !stableVersionPattern.MatchString(strings.TrimSpace(release.Version)) {
		return Release{}, fmt.Errorf("invalid release version %q; updater accepts stable x.y.z releases only", release.Version)
	}
	version, ok := normalizeVersion(release.Version)
	if !ok {
		return Release{}, fmt.Errorf("invalid release version %q", release.Version)
	}
	release.Version = version

	wantFilename := artifactFilename(s.GOOS, s.GOARCH)
	if release.Artifact.OS != s.GOOS || release.Artifact.Arch != s.GOARCH || release.Artifact.Filename != wantFilename {
		return Release{}, fmt.Errorf("release artifact does not match %s/%s", s.GOOS, s.GOARCH)
	}
	if !sha256Pattern.MatchString(release.Artifact.SHA256) {
		return Release{}, errors.New("release artifact has an invalid SHA-256 digest")
	}
	release.Artifact.SHA256 = strings.ToLower(release.Artifact.SHA256)

	parsed, err := url.Parse(release.Artifact.URL)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && !s.AllowHTTP) {
		return Release{}, errors.New("release artifact has an invalid download URL")
	}
	if path.Base(parsed.Path) != wantFilename {
		return Release{}, fmt.Errorf("release artifact URL does not end in %s", wantFilename)
	}
	if !strings.Contains(parsed.EscapedPath(), "/v"+release.Version+"/") {
		return Release{}, fmt.Errorf("release artifact URL does not contain version v%s", release.Version)
	}
	if !s.AllowHTTP {
		trustedArtifactPath := trustedArtifactRoot + "v" + release.Version + "/" + wantFilename
		if parsed.Host != trustedArtifactHost || parsed.EscapedPath() != trustedArtifactPath {
			return Release{}, fmt.Errorf("release artifact URL %q is not trusted", release.Artifact.URL)
		}
	}

	return release, nil
}

func StatusFor(current string, release Release) (Status, error) {
	currentVersion, ok := normalizeVersion(current)
	if !ok {
		return Status{}, ErrDevelopmentBuild
	}
	latestVersion, ok := normalizeVersion(release.Version)
	if !ok {
		return Status{}, fmt.Errorf("invalid latest version %q", release.Version)
	}

	return Status{
		CurrentVersion:  currentVersion,
		LatestVersion:   latestVersion,
		UpdateAvailable: semver.Compare("v"+latestVersion, "v"+currentVersion) > 0,
		Source:          release.Source,
	}, nil
}

func normalizeVersion(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "dev" || value == "(devel)" {
		return "", false
	}
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	canonical := semver.Canonical(value)
	if canonical == "" {
		return "", false
	}
	return strings.TrimPrefix(canonical, "v"), true
}

func supportedPlatform(goos, goarch string) bool {
	return (goos == "darwin" || goos == "linux") && (goarch == "amd64" || goarch == "arm64")
}

func artifactFilename(goos, goarch string) string {
	return fmt.Sprintf("lctl-%s-%s.tar.gz", goos, goarch)
}
