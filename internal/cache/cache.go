// Package cache provides skill caching functionality for improved performance.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

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
// If cacheDir is empty, defaults to ~/.skillsync/cache
func New(sourceName string, cacheDir string) (*Cache, error) {
	if cacheDir == "" {
		cacheDir = filepath.Join(util.SkillsyncConfigPath(), "cache")
	}
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
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
			cache.Entries = make(map[string]Entry)
		}
		// Version mismatch, invalidate cache
		if cache.Version != cacheVersion {
			cache.Entries = make(map[string]Entry)
			cache.Version = cacheVersion
		}
	}

	cache.path = cachePath
	return cache, nil
}

// Get retrieves a cached skill if it exists and is still valid
func (c *Cache) Get(key string) (model.Skill, bool) {
	entry, exists := c.Entries[key]
	if !exists {
		return model.Skill{}, false
	}

	// Check if source file has been modified
	if info, err := os.Stat(entry.SourcePath); err == nil {
		if info.ModTime().After(entry.SourceMod) {
			// Source has been modified, cache is stale
			delete(c.Entries, key)
			return model.Skill{}, false
		}
	}

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
}

// Save persists the cache to disk
func (c *Cache) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	// #nosec G306 - cache files should be readable by user
	return os.WriteFile(c.path, data, 0o644)
}

// Clear removes all entries from the cache
func (c *Cache) Clear() error {
	c.Entries = make(map[string]Entry)
	return os.Remove(c.path)
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
	pruned := 0
	for key, entry := range c.Entries {
		if time.Since(entry.CachedAt) > ttl {
			delete(c.Entries, key)
			pruned++
		}
	}
	return pruned
}
