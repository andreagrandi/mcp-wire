package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	cacheDirName  = "mcp-wire"
	cacheFileName = "servers.json"
	syncPageLimit = 100
)

// SyncMode identifies the type of cache sync currently in progress.
type SyncMode string

const (
	SyncModeCold        SyncMode = "cold"
	SyncModeIncremental SyncMode = "incremental"
)

// SyncProgress reports cache sync progress for UI/status updates.
type SyncProgress struct {
	Mode    SyncMode
	Pages   int
	Fetched int
	Updated int
	Cached  int
}

// SyncProgressCallback receives sync progress and a snapshot of cached servers.
type SyncProgressCallback func(progress SyncProgress, snapshot []ServerResponse)

// ServerLister abstracts the registry client for testability.
type ServerLister interface {
	ListServers(opts ListOptions) (*ServerListResponse, error)
}

// CacheStore is the on-disk cache format.
type CacheStore struct {
	LastSynced time.Time        `json:"last_synced"`
	Servers    []ServerResponse `json:"servers"`
}

// Cache provides local caching and in-memory search over registry servers.
type Cache struct {
	path   string
	client ServerLister
	store  CacheStore
	onSync SyncProgressCallback
}

// NewCache creates a cache backed by the default cache path.
func NewCache(client ServerLister) *Cache {
	return NewCacheWithPath(client, defaultCachePath())
}

// NewCacheWithPath creates a cache at a specific file path.
func NewCacheWithPath(client ServerLister, path string) *Cache {
	return &Cache{
		path:   path,
		client: client,
	}
}

// SetSyncProgressCallback registers a callback that receives sync progress updates.
func (c *Cache) SetSyncProgressCallback(callback SyncProgressCallback) {
	c.onSync = callback
}

// Load reads the cache from disk into memory.
//
// If the file does not exist, the cache starts empty.
func (c *Cache) Load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.store = CacheStore{}
			return nil
		}

		return fmt.Errorf("read cache file %q: %w", c.path, err)
	}

	var store CacheStore
	if err := json.Unmarshal(data, &store); err != nil {
		c.store = CacheStore{}
		return nil
	}

	c.store = store
	return nil
}

// Sync fetches servers from the registry and updates the local cache.
//
// If the cache has been synced before, an incremental sync is attempted
// using updated_since. Otherwise a full paginated sync is performed.
// On network failure, the stale cache is preserved and the error is returned.
func (c *Cache) Sync() error {
	if c.store.LastSynced.IsZero() {
		return c.coldSync()
	}

	return c.incrementalSync()
}

// All returns every cached server.
func (c *Cache) All() []ServerResponse {
	result := make([]ServerResponse, len(c.store.Servers))
	copy(result, c.store.Servers)

	return result
}

// Search filters cached servers by case-insensitive substring match
// against name, title, and description.
func (c *Cache) Search(query string) []ServerResponse {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return c.All()
	}

	var matches []ServerResponse

	for _, srv := range c.store.Servers {
		if strings.Contains(strings.ToLower(srv.Server.Name), q) ||
			strings.Contains(strings.ToLower(srv.Server.Title), q) ||
			strings.Contains(strings.ToLower(srv.Server.Description), q) {
			matches = append(matches, srv)
		}
	}

	return matches
}

// LastSynced returns the timestamp of the last successful sync.
func (c *Cache) LastSynced() time.Time {
	return c.store.LastSynced
}

// Count returns the number of cached servers.
func (c *Cache) Count() int {
	return len(c.store.Servers)
}

func (c *Cache) coldSync() error {
	var all []ServerResponse
	cursor := ""
	pages := 0

	for {
		resp, err := c.client.ListServers(ListOptions{
			Limit:  syncPageLimit,
			Cursor: cursor,
		})
		if err != nil {
			return fmt.Errorf("cold sync: %w", err)
		}

		all = append(all, resp.Servers...)
		pages++
		c.store.Servers = all
		c.emitSyncProgress(SyncProgress{
			Mode:    SyncModeCold,
			Pages:   pages,
			Fetched: len(all),
			Cached:  len(c.store.Servers),
		})

		if resp.Metadata.NextCursor == "" {
			break
		}

		cursor = resp.Metadata.NextCursor
	}

	c.store.Servers = all
	c.store.LastSynced = time.Now().UTC()
	c.emitSyncProgress(SyncProgress{
		Mode:    SyncModeCold,
		Pages:   pages,
		Fetched: len(all),
		Cached:  len(c.store.Servers),
	})

	return c.save()
}

func (c *Cache) incrementalSync() error {
	since := c.store.LastSynced.Format(time.RFC3339)
	index := c.buildIndex()

	cursor := ""
	pages := 0
	updatedCount := 0

	for {
		resp, err := c.client.ListServers(ListOptions{
			Limit:        syncPageLimit,
			Cursor:       cursor,
			UpdatedSince: since,
		})
		if err != nil {
			return fmt.Errorf("incremental sync: %w", err)
		}

		for _, updated := range resp.Servers {
			if i, ok := index[updated.Server.Name]; ok {
				c.store.Servers[i] = updated
			} else {
				c.store.Servers = append(c.store.Servers, updated)
				index[updated.Server.Name] = len(c.store.Servers) - 1
			}
			updatedCount++
		}

		pages++
		c.emitSyncProgress(SyncProgress{
			Mode:    SyncModeIncremental,
			Pages:   pages,
			Updated: updatedCount,
			Cached:  len(c.store.Servers),
		})

		if resp.Metadata.NextCursor == "" {
			break
		}

		cursor = resp.Metadata.NextCursor
	}

	c.store.LastSynced = time.Now().UTC()
	c.emitSyncProgress(SyncProgress{
		Mode:    SyncModeIncremental,
		Pages:   pages,
		Updated: updatedCount,
		Cached:  len(c.store.Servers),
	})

	return c.save()
}

func (c *Cache) emitSyncProgress(progress SyncProgress) {
	if c.onSync == nil {
		return
	}

	snapshot := make([]ServerResponse, len(c.store.Servers))
	copy(snapshot, c.store.Servers)

	c.onSync(progress, snapshot)
}

func (c *Cache) buildIndex() map[string]int {
	index := make(map[string]int, len(c.store.Servers))
	for i, srv := range c.store.Servers {
		index[srv.Server.Name] = i
	}

	return index
}

func (c *Cache) save() error {
	cacheDir := filepath.Dir(c.path)
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return fmt.Errorf("create cache directory %q: %w", cacheDir, err)
	}

	data, err := json.Marshal(c.store)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0o644); err != nil {
		return fmt.Errorf("write cache file %q: %w", c.path, err)
	}

	return nil
}

// DefaultCachePath returns the on-disk path of the registry cache file.
func DefaultCachePath() string {
	return defaultCachePath()
}

// ClearDefaultCache removes the on-disk registry cache file.
// It returns the cache path and whether a file was removed.
func ClearDefaultCache() (string, bool, error) {
	path := defaultCachePath()

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return path, false, nil
		}

		return path, false, fmt.Errorf("remove cache file %q: %w", path, err)
	}

	// Best-effort: remove directory if empty.
	_ = os.Remove(filepath.Dir(path))

	return path, true, nil
}

func defaultCachePath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(".cache", cacheDirName, cacheFileName)
	}

	return filepath.Join(cacheDir, cacheDirName, cacheFileName)
}
