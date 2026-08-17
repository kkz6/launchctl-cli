package selfupdate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheRoundTripUsesPrivateAtomicFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cache")
	cachePath := filepath.Join(root, "update.json")
	now := time.Date(2026, time.August, 17, 12, 30, 0, 0, time.FixedZone("IST", 5*60*60+30*60))
	cache := Cache{Path: cachePath, Now: func() time.Time { return now }}
	release := testRelease("1.2.3", testDigest, "https://sin1.contabostorage.com/bucket/v1.2.3/lctl-linux-amd64.tar.gz")

	if err := cache.RecordSuccess(release); err != nil {
		t.Fatal(err)
	}
	status, found, err := cache.CachedStatus("1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !found || !status.UpdateAvailable || status.LatestVersion != "1.2.3" {
		t.Fatalf("unexpected cached status: found=%t status=%+v", found, status)
	}
	if status.CheckedAt != now.UTC().Format(time.RFC3339) {
		t.Fatalf("CheckedAt = %q, want %q", status.CheckedAt, now.UTC().Format(time.RFC3339))
	}

	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("cache permissions = %04o, want 0600", got)
	}
	dirInfo, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got&0077 != 0 {
		t.Fatalf("cache directory permissions = %04o, want no group/other access", got)
	}

	matches, err := filepath.Glob(filepath.Join(root, ".update-cache-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("cache staging files were not removed: %v", matches)
	}
}

func TestCacheRefreshTTLs(t *testing.T) {
	now := time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)
	cache := Cache{
		Path:       filepath.Join(t.TempDir(), "update.json"),
		Now:        func() time.Time { return now },
		SuccessTTL: 2 * time.Hour,
		FailureTTL: 15 * time.Minute,
	}
	release := testRelease("1.2.3", testDigest, "https://sin1.contabostorage.com/bucket/v1.2.3/lctl-linux-amd64.tar.gz")

	if err := cache.RecordSuccess(release); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2*time.Hour - time.Second)
	if cache.NeedsRefresh() {
		t.Fatal("fresh successful cache needs refresh")
	}
	now = now.Add(time.Second)
	if !cache.NeedsRefresh() {
		t.Fatal("expired successful cache does not need refresh")
	}

	if err := cache.RecordFailure(os.ErrDeadlineExceeded); err != nil {
		t.Fatal(err)
	}
	now = now.Add(15*time.Minute - time.Second)
	if cache.NeedsRefresh() {
		t.Fatal("fresh failure backoff needs refresh")
	}
	now = now.Add(time.Second)
	if !cache.NeedsRefresh() {
		t.Fatal("expired failure backoff does not need refresh")
	}

	status, found, err := cache.CachedStatus("1.0.0")
	if err != nil || !found || status.LatestVersion != "1.2.3" {
		t.Fatalf("failure discarded the last successful release: found=%t status=%+v err=%v", found, status, err)
	}
}

func TestCacheRefreshesAfterClockMovesBack(t *testing.T) {
	writeTime := time.Date(2026, time.August, 18, 0, 0, 0, 0, time.UTC)
	now := writeTime
	cache := Cache{Path: filepath.Join(t.TempDir(), "update.json"), Now: func() time.Time { return now }}
	if err := cache.RecordSuccess(testRelease("1.2.3", testDigest, "https://sin1.contabostorage.com/bucket/v1.2.3/lctl-linux-amd64.tar.gz")); err != nil {
		t.Fatal(err)
	}

	now = writeTime.Add(-48 * time.Hour)
	if !cache.NeedsRefresh() {
		t.Fatal("a cache timestamp in the future suppressed refresh indefinitely")
	}
}

func TestCacheRejectsMalformedUnsupportedAndOversizedFiles(t *testing.T) {
	tests := map[string][]byte{
		"malformed": []byte("not json"),
		"wrong schema": mustJSON(t, cacheFile{
			SchemaVersion: cacheSchemaVersion + 1,
			LastAttemptAt: time.Now(),
		}),
		"oversized": []byte(strings.Repeat("x", maxCacheBytes+1)),
	}

	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "update.json")
			if err := os.WriteFile(path, contents, 0600); err != nil {
				t.Fatal(err)
			}
			cache := Cache{Path: path}
			if _, _, err := cache.CachedStatus("1.0.0"); err == nil {
				t.Fatal("CachedStatus accepted an invalid cache file")
			}
			if !cache.NeedsRefresh() {
				t.Fatal("invalid cache did not request a refresh")
			}
		})
	}
}

func TestCacheMissingAndFailureOnlyAreNotStatuses(t *testing.T) {
	cache := Cache{Path: filepath.Join(t.TempDir(), "update.json")}
	if _, found, err := cache.CachedStatus("1.0.0"); err != nil || found {
		t.Fatalf("missing cache: found=%t err=%v", found, err)
	}
	if err := cache.RecordFailure(os.ErrDeadlineExceeded); err != nil {
		t.Fatal(err)
	}
	if _, found, err := cache.CachedStatus("1.0.0"); err != nil || found {
		t.Fatalf("failure-only cache: found=%t err=%v", found, err)
	}
}

func TestRefreshLockIsExclusiveAndReleasable(t *testing.T) {
	cache := Cache{Path: filepath.Join(t.TempDir(), "update.json")}
	release, acquired, err := cache.TryRefreshLock()
	if err != nil || !acquired {
		t.Fatalf("first lock: acquired=%t err=%v", acquired, err)
	}
	if _, acquired, err := cache.TryRefreshLock(); err != nil || acquired {
		t.Fatalf("second lock: acquired=%t err=%v", acquired, err)
	}

	release()
	release() // Release closures are intentionally idempotent.
	thirdRelease, acquired, err := cache.TryRefreshLock()
	if err != nil || !acquired {
		t.Fatalf("lock after release: acquired=%t err=%v", acquired, err)
	}
	thirdRelease()
}

func TestReleaseRefreshLockSupportsBackgroundOwner(t *testing.T) {
	cache := Cache{Path: filepath.Join(t.TempDir(), "update.json")}
	_, acquired, err := cache.TryRefreshLock()
	if err != nil || !acquired {
		t.Fatalf("TryRefreshLock: acquired=%t err=%v", acquired, err)
	}

	cache.ReleaseRefreshLock()
	release, acquired, err := cache.TryRefreshLock()
	if err != nil || !acquired {
		t.Fatalf("lock after background release: acquired=%t err=%v", acquired, err)
	}
	release()
}

func TestRefreshLockRecoversStaleLock(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	cache := Cache{Path: filepath.Join(t.TempDir(), "update.json"), Now: func() time.Time { return now }}
	lockPath := cache.Path + ".lock"
	if err := os.WriteFile(lockPath, []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	stale := now.Add(-refreshLockTTL - time.Second)
	if err := os.Chtimes(lockPath, stale, stale); err != nil {
		t.Fatal(err)
	}

	release, acquired, err := cache.TryRefreshLock()
	if err != nil || !acquired {
		t.Fatalf("stale lock recovery: acquired=%t err=%v", acquired, err)
	}
	release()
}

func TestRefreshLockRecoversAfterClockMovesBack(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	cache := Cache{Path: filepath.Join(t.TempDir(), "update.json"), Now: func() time.Time { return now }}
	lockPath := cache.Path + ".lock"
	if err := os.WriteFile(lockPath, []byte("future"), 0600); err != nil {
		t.Fatal(err)
	}
	future := now.Add(48 * time.Hour)
	if err := os.Chtimes(lockPath, future, future); err != nil {
		t.Fatal(err)
	}

	release, acquired, err := cache.TryRefreshLock()
	if err != nil || !acquired {
		t.Fatalf("future lock recovery: acquired=%t err=%v", acquired, err)
	}
	release()
}

func TestRefreshLockHasSingleConcurrentWinner(t *testing.T) {
	cache := Cache{Path: filepath.Join(t.TempDir(), "update.json")}
	const contenders = 16
	start := make(chan struct{})
	releases := make(chan func(), contenders)
	var winners atomic.Int32
	var wait sync.WaitGroup

	for range contenders {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			release, acquired, err := cache.TryRefreshLock()
			if err != nil {
				t.Errorf("TryRefreshLock: %v", err)
				return
			}
			if acquired {
				winners.Add(1)
				releases <- release
			}
		}()
	}
	close(start)
	wait.Wait()
	close(releases)

	if got := winners.Load(); got != 1 {
		t.Fatalf("concurrent lock winners = %d, want 1", got)
	}
	for release := range releases {
		release()
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func testRelease(version, digest, artifactURL string) Release {
	return Release{
		Version: version,
		Artifact: Artifact{
			OS:       "linux",
			Arch:     "amd64",
			Filename: "lctl-linux-amd64.tar.gz",
			URL:      artifactURL,
			SHA256:   digest,
		},
		Source: "test",
	}
}
