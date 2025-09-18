package photo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	metadataCacheFileName = "photo_metadata_cache.json"
	metadataCacheVersion  = 1
)

type metadataCache struct {
	Version int                           `json:"version"`
	Entries map[string]metadataCacheEntry `json:"entries"`
}

type metadataCacheEntry struct {
	ModTime     int64     `json:"modTime"`
	TakenTime   time.Time `json:"takenTime"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Orientation int       `json:"orientation"`
}

func loadMetadataCache() (*metadataCache, error) {
	path, err := metadataCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return newMetadataCache(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read metadata cache: %w", err)
	}

	cache := newMetadataCache()
	if err := json.Unmarshal(data, cache); err != nil {
		return nil, fmt.Errorf("unmarshal metadata cache: %w", err)
	}

	if cache.Version != metadataCacheVersion || cache.Entries == nil {
		return newMetadataCache(), nil
	}

	return cache, nil
}

func saveMetadataCache(cache *metadataCache) error {
	path, err := metadataCachePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	tmpPath := path + ".tmp"
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata cache: %w", err)
	}

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write metadata cache: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace metadata cache: %w", err)
	}

	return nil
}

func metadataCachePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine user home: %w", err)
	}
	return filepath.Join(homeDir, configDirName, metadataCacheFileName), nil
}

func newMetadataCache() *metadataCache {
	return &metadataCache{
		Version: metadataCacheVersion,
		Entries: make(map[string]metadataCacheEntry),
	}
}

func (c *metadataCache) get(path string, modTime time.Time) (Photo, bool) {
	if c == nil {
		return Photo{}, false
	}
	entry, ok := c.Entries[path]
	if !ok || entry.ModTime != modTime.UnixNano() {
		return Photo{}, false
	}
	return Photo{
		FilePath:    path,
		TakenTime:   entry.TakenTime,
		Width:       entry.Width,
		Height:      entry.Height,
		Orientation: entry.Orientation,
	}, true
}

func (c *metadataCache) set(path string, modTime time.Time, photo Photo) {
	if c == nil {
		return
	}
	c.Entries[path] = metadataCacheEntry{
		ModTime:     modTime.UnixNano(),
		TakenTime:   photo.TakenTime,
		Width:       photo.Width,
		Height:      photo.Height,
		Orientation: photo.Orientation,
	}
}

func (c *metadataCache) prune(validPaths map[string]struct{}) bool {
	if c == nil {
		return false
	}
	changed := false
	for path := range c.Entries {
		if _, ok := validPaths[path]; !ok {
			delete(c.Entries, path)
			changed = true
		}
	}
	return changed
}

const configDirName = ".openframe"
