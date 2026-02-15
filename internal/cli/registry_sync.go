package cli

import (
	"fmt"
	"sync"

	"github.com/andreagrandi/mcp-wire/internal/registry"
)

type registrySyncState struct {
	mu sync.RWMutex

	once sync.Once

	started bool
	syncing bool

	mode    registry.SyncMode
	pages   int
	fetched int
	updated int
	cached  int

	err error

	servers []registry.ServerResponse
}

var backgroundRegistrySync registrySyncState

func maybeStartRegistryBackgroundSync() {
	cfg, err := loadConfig()
	if err != nil || !cfg.IsFeatureEnabled("registry") {
		return
	}

	ensureRegistrySyncStarted(true)
}

func ensureRegistrySyncStarted(registryEnabled bool) {
	if !registryEnabled {
		return
	}

	backgroundRegistrySync.once.Do(func() {
		seed := registry.NewCache(nil)
		if err := seed.Load(); err == nil {
			snapshot := seed.All()
			backgroundRegistrySync.mu.Lock()
			backgroundRegistrySync.servers = snapshot
			backgroundRegistrySync.cached = len(snapshot)
			backgroundRegistrySync.mu.Unlock()
		}

		backgroundRegistrySync.mu.Lock()
		backgroundRegistrySync.started = true
		backgroundRegistrySync.syncing = true
		backgroundRegistrySync.mu.Unlock()

		go runRegistryBackgroundSync()
	})
}

func runRegistryBackgroundSync() {
	cache := registry.NewCache(registry.NewClient())
	cache.SetSyncProgressCallback(func(progress registry.SyncProgress, snapshot []registry.ServerResponse) {
		backgroundRegistrySync.mu.Lock()
		defer backgroundRegistrySync.mu.Unlock()

		backgroundRegistrySync.syncing = true
		backgroundRegistrySync.mode = progress.Mode
		backgroundRegistrySync.pages = progress.Pages
		backgroundRegistrySync.fetched = progress.Fetched
		backgroundRegistrySync.updated = progress.Updated
		backgroundRegistrySync.cached = progress.Cached
		backgroundRegistrySync.servers = snapshot
		backgroundRegistrySync.err = nil
	})

	if err := cache.Load(); err == nil {
		snapshot := cache.All()
		backgroundRegistrySync.mu.Lock()
		backgroundRegistrySync.servers = snapshot
		backgroundRegistrySync.cached = len(snapshot)
		backgroundRegistrySync.mu.Unlock()
	}

	syncErr := cache.Sync()
	finalSnapshot := cache.All()

	backgroundRegistrySync.mu.Lock()
	backgroundRegistrySync.servers = finalSnapshot
	backgroundRegistrySync.cached = len(finalSnapshot)
	backgroundRegistrySync.syncing = false
	backgroundRegistrySync.err = syncErr
	backgroundRegistrySync.mu.Unlock()
}

func loadRegistryServersSnapshot() []registry.ServerResponse {
	backgroundRegistrySync.mu.RLock()
	started := backgroundRegistrySync.started
	servers := make([]registry.ServerResponse, len(backgroundRegistrySync.servers))
	copy(servers, backgroundRegistrySync.servers)
	backgroundRegistrySync.mu.RUnlock()

	if started {
		return servers
	}

	cache := registry.NewCache(nil)
	if err := cache.Load(); err != nil {
		return nil
	}

	return cache.All()
}

func registrySyncStatusLine(registryEnabled bool) string {
	if !registryEnabled {
		return ""
	}

	backgroundRegistrySync.mu.RLock()
	started := backgroundRegistrySync.started
	syncing := backgroundRegistrySync.syncing
	mode := backgroundRegistrySync.mode
	fetched := backgroundRegistrySync.fetched
	updated := backgroundRegistrySync.updated
	cached := backgroundRegistrySync.cached
	err := backgroundRegistrySync.err
	backgroundRegistrySync.mu.RUnlock()

	if !started {
		return ""
	}

	if syncing {
		if mode == registry.SyncModeIncremental {
			if updated > 0 {
				return fmt.Sprintf("Registry sync in background (%d updates, %d cached)", updated, cached)
			}

			return fmt.Sprintf("Registry sync in background (%d cached)", cached)
		}

		if fetched > 0 {
			return fmt.Sprintf("Registry sync in background (%d+ servers fetched so far)", fetched)
		}

		return "Registry sync in background"
	}

	if err != nil {
		return fmt.Sprintf("Registry sync failed; using cached results (%d servers)", cached)
	}

	return ""
}
