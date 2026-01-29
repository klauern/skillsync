// Package cache provides skill caching functionality for improved performance.
package cache

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/util"
)

// Entry represents a cached skill entry with metadata
type Entry struct {
	Skill      model.Skill `json:"skill"`
	CachedAt   time.Time   `json:"cached_at"`
	SourcePath string      `json:"source_path"`
	SourceMod  time.Time   `json:"source_mod"`
}

// Cache manages cached skills for a specific source type
type Cache struct {
	Version string           `json:"version"`
	Entries map[string]Entry `json:"entries"`
	path    string
}

const (
	cacheVersion = "1.0"
	// DefaultTTL is the default time-to-live for cache entries
	DefaultTTL = 1 * time.Hour
)

// New creates or loads a cache for the given source name (e.g., "plugins")
func New(sourceName string) (*Cache, error) {
	logging.Debug("initializing cache",
		slog.String("source", sourceName),
		logging.Operation("cache_init"),
	)

	cacheDir := filepath.Join(util.SkillsyncConfigPath(), "cache")
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		logging.Error("failed to create cache directory",
			logging.Path(cacheDir),
			logging.Err(err),
		)
		return nil, err
	}

	cachePath := filepath.Join(cacheDir, sourceName+".json")
	cache := &Cache{
		Version: cacheVersion,
		Entries: make(map[string]Entry),
		path:    cachePath,
	}

	// Try to load existing cache
	// #nosec G304 - cachePath is constructed from trusted configuration path
	if data, err := os.ReadFile(cachePath); err == nil {
		if err := json.Unmarshal(data, cache); err != nil {
			// Corrupted cache, start fresh
			logging.Warn("corrupted cache, starting fresh",
				slog.String("source", sourceName),
				logging.Path(cachePath),
				logging.Err(err),
			)
			cache.Entries = make(map[string]Entry)
		}
		// Version mismatch, invalidate cache
		if cache.Version != cacheVersion {
			logging.Debug("cache version mismatch, invalidating",
				slog.String("source", sourceName),
				slog.String("expected", cacheVersion),
				slog.String("actual", cache.Version),
			)
			cache.Entries = make(map[string]Entry)
			cache.Version = cacheVersion
		} else {
			logging.Debug("cache loaded",
				slog.String("source", sourceName),
				logging.Count(len(cache.Entries)),
			)
		}
	} else {
		logging.Debug("no existing cache found, creating new",
			slog.String("source", sourceName),
		)
	}

	cache.path = cachePath
	return cache, nil
}

// Get retrieves a cached skill if it exists and is still valid
func (c *Cache) Get(key string) (model.Skill, bool) {
	entry, exists := c.Entries[key]
	if !exists {
		logging.Debug("cache miss", slog.String("key", key))
		return model.Skill{}, false
	}

	// Check if source file has been modified
	if info, err := os.Stat(entry.SourcePath); err == nil {
		if info.ModTime().After(entry.SourceMod) {
			// Source has been modified, cache is stale
			logging.Debug("cache stale (source modified)",
				slog.String("key", key),
				logging.Path(entry.SourcePath),
			)
			delete(c.Entries, key)
			return model.Skill{}, false
		}
	}

	logging.Debug("cache hit", slog.String("key", key))
	return entry.Skill, true
}

// Set stores a skill in the cache
func (c *Cache) Set(key string, skill model.Skill) {
	sourceMod := time.Now()
	if info, err := os.Stat(skill.Path); err == nil {
		sourceMod = info.ModTime()
	}

	c.Entries[key] = Entry{
		Skill:      skill,
		CachedAt:   time.Now(),
		SourcePath: skill.Path,
		SourceMod:  sourceMod,
	}

	logging.Debug("cache set",
		slog.String("key", key),
		logging.Skill(skill.Name),
		logging.Path(skill.Path),
	)
}

// Save persists the cache to disk
func (c *Cache) Save() error {
	logging.Debug("saving cache",
		logging.Path(c.path),
		logging.Count(len(c.Entries)),
	)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		logging.Error("failed to marshal cache", logging.Err(err))
		return err
	}

	// #nosec G306 - cache files should be readable by user
	if err := os.WriteFile(c.path, data, 0o644); err != nil {
		logging.Error("failed to write cache file",
			logging.Path(c.path),
			logging.Err(err),
		)
		return err
	}

	logging.Debug("cache saved successfully", logging.Path(c.path))
	return nil
}

// Clear removes all entries from the cache
func (c *Cache) Clear() error {
	logging.Info("clearing cache",
		logging.Path(c.path),
		logging.Count(len(c.Entries)),
	)

	c.Entries = make(map[string]Entry)
	if err := os.Remove(c.path); err != nil && !os.IsNotExist(err) {
		logging.Error("failed to remove cache file",
			logging.Path(c.path),
			logging.Err(err),
		)
		return err
	}

	return nil
}

// Size returns the number of entries in the cache
func (c *Cache) Size() int {
	return len(c.Entries)
}

// IsStale checks if any cache entry has expired based on TTL
func (c *Cache) IsStale(ttl time.Duration) bool {
	for _, entry := range c.Entries {
		if time.Since(entry.CachedAt) > ttl {
			return true
		}
	}
	return false
}

// Prune removes stale entries based on TTL
func (c *Cache) Prune(ttl time.Duration) int {
	logging.Debug("pruning cache",
		slog.Duration("ttl", ttl),
		logging.Count(len(c.Entries)),
	)

	pruned := 0
	for key, entry := range c.Entries {
		if time.Since(entry.CachedAt) > ttl {
			delete(c.Entries, key)
			pruned++
		}
	}

	if pruned > 0 {
		logging.Info("cache pruned",
			logging.Count(pruned),
			slog.Int("remaining", len(c.Entries)),
		)
	} else {
		logging.Debug("no stale entries to prune")
	}

	return pruned
}
