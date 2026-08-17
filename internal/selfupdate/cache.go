package selfupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	cacheSchemaVersion = 1
	maxCacheBytes      = 64 << 10
	defaultSuccessTTL  = 24 * time.Hour
	defaultFailureTTL  = time.Hour
	refreshLockTTL     = 10 * time.Minute
)

type cacheFile struct {
	SchemaVersion int       `json:"schema_version"`
	CheckedAt     time.Time `json:"checked_at,omitempty"`
	LastAttemptAt time.Time `json:"last_attempt_at"`
	LastError     string    `json:"last_error,omitempty"`
	Release       Release   `json:"release,omitempty"`
}

type Cache struct {
	Path       string
	Now        func() time.Time
	SuccessTTL time.Duration
	FailureTTL time.Duration
}

func DefaultCache() (Cache, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return Cache{}, fmt.Errorf("locate user cache directory: %w", err)
	}
	return Cache{Path: filepath.Join(root, "launchctl", "update.json")}, nil
}

func (c Cache) CachedStatus(current string) (Status, bool, error) {
	cached, err := c.load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Status{}, false, nil
		}
		return Status{}, false, err
	}
	if cached.Release.Version == "" {
		return Status{}, false, nil
	}
	status, err := StatusFor(current, cached.Release)
	if err != nil {
		return Status{}, false, err
	}
	status.CheckedAt = cached.CheckedAt.UTC().Format(time.RFC3339)
	return status, true, nil
}

func (c Cache) NeedsRefresh() bool {
	cached, err := c.load()
	if err != nil {
		return true
	}

	ttl := c.successTTL()
	if cached.LastError != "" {
		ttl = c.failureTTL()
	}
	if cached.LastAttemptAt.IsZero() {
		return true
	}
	age := c.now().Sub(cached.LastAttemptAt)
	// A wall clock correction must not make a stale cache appear fresh for an
	// unbounded period.
	return age < 0 || age >= ttl
}

func (c Cache) RecordSuccess(release Release) error {
	now := c.now().UTC()
	return c.store(cacheFile{
		SchemaVersion: cacheSchemaVersion,
		CheckedAt:     now,
		LastAttemptAt: now,
		Release:       release,
	})
}

func (c Cache) RecordFailure(checkErr error) error {
	cached, err := c.load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		cached = cacheFile{}
	}
	cached.SchemaVersion = cacheSchemaVersion
	cached.LastAttemptAt = c.now().UTC()
	if checkErr != nil {
		cached.LastError = checkErr.Error()
	}
	return c.store(cached)
}

func (c Cache) TryRefreshLock() (func(), bool, error) {
	if c.Path == "" {
		return nil, false, errors.New("update cache path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(c.Path), 0700); err != nil {
		return nil, false, fmt.Errorf("create update cache directory: %w", err)
	}

	lockPath := c.Path + ".lock"
	file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil && errors.Is(err, os.ErrExist) {
		info, statErr := os.Stat(lockPath)
		age := time.Duration(0)
		if statErr == nil {
			age = c.now().Sub(info.ModTime())
		}
		if statErr == nil && (age < 0 || age >= refreshLockTTL) {
			if removeErr := os.Remove(lockPath); removeErr == nil {
				return c.TryRefreshLock()
			}
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("create update refresh lock: %w", err)
	}
	if closeErr := file.Close(); closeErr != nil {
		_ = os.Remove(lockPath)
		return nil, false, fmt.Errorf("close update refresh lock: %w", closeErr)
	}

	released := false
	return func() {
		if !released {
			released = true
			_ = os.Remove(lockPath)
		}
	}, true, nil
}

func (c Cache) ReleaseRefreshLock() {
	if c.Path != "" {
		_ = os.Remove(c.Path + ".lock")
	}
}

func (c Cache) load() (cacheFile, error) {
	if c.Path == "" {
		return cacheFile{}, errors.New("update cache path is empty")
	}
	file, err := os.Open(c.Path)
	if err != nil {
		return cacheFile{}, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxCacheBytes+1))
	if err != nil {
		return cacheFile{}, fmt.Errorf("read update cache: %w", err)
	}
	if len(data) > maxCacheBytes {
		return cacheFile{}, errors.New("update cache exceeds the size limit")
	}

	var cached cacheFile
	if err := json.Unmarshal(data, &cached); err != nil {
		return cacheFile{}, fmt.Errorf("decode update cache: %w", err)
	}
	if cached.SchemaVersion != cacheSchemaVersion {
		return cacheFile{}, fmt.Errorf("unsupported update cache schema %d", cached.SchemaVersion)
	}
	return cached, nil
}

func (c Cache) store(cached cacheFile) error {
	if c.Path == "" {
		return errors.New("update cache path is empty")
	}
	root := filepath.Dir(c.Path)
	if err := os.MkdirAll(root, 0700); err != nil {
		return fmt.Errorf("create update cache directory: %w", err)
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("encode update cache: %w", err)
	}
	data = append(data, '\n')

	temporary, err := os.CreateTemp(root, ".update-cache-*")
	if err != nil {
		return fmt.Errorf("create update cache staging file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set update cache permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write update cache: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync update cache: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close update cache: %w", err)
	}
	if err := os.Rename(temporaryPath, c.Path); err != nil {
		return fmt.Errorf("activate update cache: %w", err)
	}
	if err := syncDirectory(root); err != nil {
		return fmt.Errorf("sync update cache directory: %w", err)
	}
	return nil
}

func (c Cache) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c Cache) successTTL() time.Duration {
	if c.SuccessTTL > 0 {
		return c.SuccessTTL
	}
	return defaultSuccessTTL
}

func (c Cache) failureTTL() time.Duration {
	if c.FailureTTL > 0 {
		return c.FailureTTL
	}
	return defaultFailureTTL
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
