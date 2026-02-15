package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mockLister is a test double for ServerLister.
type mockLister struct {
	pages []ServerListResponse
	calls []ListOptions
	err   error
}

func (m *mockLister) ListServers(opts ListOptions) (*ServerListResponse, error) {
	m.calls = append(m.calls, opts)
	if m.err != nil {
		return nil, m.err
	}

	idx := len(m.calls) - 1
	if idx >= len(m.pages) {
		return &ServerListResponse{Metadata: Metadata{Count: 0}}, nil
	}

	return &m.pages[idx], nil
}

func sampleServer(name string, desc string) ServerResponse {
	return ServerResponse{
		Server: ServerJSON{
			Name:        name,
			Description: desc,
			Version:     "1.0.0",
		},
	}
}

func TestColdSyncFetchesAllPages(t *testing.T) {
	mock := &mockLister{
		pages: []ServerListResponse{
			{
				Servers:  []ServerResponse{sampleServer("ns/server-a", "First server")},
				Metadata: Metadata{Count: 1, NextCursor: "page2"},
			},
			{
				Servers:  []ServerResponse{sampleServer("ns/server-b", "Second server")},
				Metadata: Metadata{Count: 1},
			},
		},
	}

	cache := NewCacheWithPath(mock, filepath.Join(t.TempDir(), "servers.json"))

	if err := cache.Sync(); err != nil {
		t.Fatalf("expected sync to succeed: %v", err)
	}

	if cache.Count() != 2 {
		t.Fatalf("expected 2 servers, got %d", cache.Count())
	}

	if len(mock.calls) != 2 {
		t.Fatalf("expected 2 API calls, got %d", len(mock.calls))
	}

	if mock.calls[0].Cursor != "" {
		t.Fatalf("expected empty cursor on first call, got %q", mock.calls[0].Cursor)
	}

	if mock.calls[1].Cursor != "page2" {
		t.Fatalf("expected cursor=page2 on second call, got %q", mock.calls[1].Cursor)
	}
}

func TestColdSyncUsesMaxLimit(t *testing.T) {
	mock := &mockLister{
		pages: []ServerListResponse{
			{Servers: []ServerResponse{}, Metadata: Metadata{Count: 0}},
		},
	}

	cache := NewCacheWithPath(mock, filepath.Join(t.TempDir(), "servers.json"))

	if err := cache.Sync(); err != nil {
		t.Fatalf("expected sync to succeed: %v", err)
	}

	if mock.calls[0].Limit != syncPageLimit {
		t.Fatalf("expected limit=%d, got %d", syncPageLimit, mock.calls[0].Limit)
	}
}

func TestIncrementalSyncUsesUpdatedSince(t *testing.T) {
	mock := &mockLister{
		pages: []ServerListResponse{
			{Servers: []ServerResponse{}, Metadata: Metadata{Count: 0}},
		},
	}

	cache := NewCacheWithPath(mock, filepath.Join(t.TempDir(), "servers.json"))
	cache.store.LastSynced = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	cache.store.Servers = []ServerResponse{sampleServer("ns/existing", "Existing")}

	if err := cache.Sync(); err != nil {
		t.Fatalf("expected sync to succeed: %v", err)
	}

	if mock.calls[0].UpdatedSince != "2025-06-01T00:00:00Z" {
		t.Fatalf("expected updated_since=2025-06-01T00:00:00Z, got %q", mock.calls[0].UpdatedSince)
	}
}

func TestIncrementalSyncUpdatesExistingServer(t *testing.T) {
	mock := &mockLister{
		pages: []ServerListResponse{
			{
				Servers: []ServerResponse{
					{Server: ServerJSON{Name: "ns/server-a", Description: "Updated desc", Version: "2.0.0"}},
				},
				Metadata: Metadata{Count: 1},
			},
		},
	}

	cache := NewCacheWithPath(mock, filepath.Join(t.TempDir(), "servers.json"))
	cache.store.LastSynced = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cache.store.Servers = []ServerResponse{
		{Server: ServerJSON{Name: "ns/server-a", Description: "Old desc", Version: "1.0.0"}},
		{Server: ServerJSON{Name: "ns/server-b", Description: "Unchanged", Version: "1.0.0"}},
	}

	if err := cache.Sync(); err != nil {
		t.Fatalf("expected sync to succeed: %v", err)
	}

	if cache.Count() != 2 {
		t.Fatalf("expected 2 servers (not duplicated), got %d", cache.Count())
	}

	all := cache.All()
	for _, srv := range all {
		if srv.Server.Name == "ns/server-a" {
			if srv.Server.Version != "2.0.0" {
				t.Fatalf("expected server-a version=2.0.0, got %q", srv.Server.Version)
			}

			if srv.Server.Description != "Updated desc" {
				t.Fatalf("expected server-a description updated, got %q", srv.Server.Description)
			}
		}
	}
}

func TestIncrementalSyncAddsNewServers(t *testing.T) {
	mock := &mockLister{
		pages: []ServerListResponse{
			{
				Servers:  []ServerResponse{sampleServer("ns/new-server", "Brand new")},
				Metadata: Metadata{Count: 1},
			},
		},
	}

	cache := NewCacheWithPath(mock, filepath.Join(t.TempDir(), "servers.json"))
	cache.store.LastSynced = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cache.store.Servers = []ServerResponse{sampleServer("ns/existing", "Existing")}

	if err := cache.Sync(); err != nil {
		t.Fatalf("expected sync to succeed: %v", err)
	}

	if cache.Count() != 2 {
		t.Fatalf("expected 2 servers, got %d", cache.Count())
	}
}

func TestSyncPersistsToDisk(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "servers.json")
	mock := &mockLister{
		pages: []ServerListResponse{
			{
				Servers:  []ServerResponse{sampleServer("ns/server-a", "A server")},
				Metadata: Metadata{Count: 1},
			},
		},
	}

	cache := NewCacheWithPath(mock, cachePath)
	if err := cache.Sync(); err != nil {
		t.Fatalf("expected sync to succeed: %v", err)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("expected cache file to exist: %v", err)
	}

	var store CacheStore
	if err := json.Unmarshal(data, &store); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}

	if len(store.Servers) != 1 {
		t.Fatalf("expected 1 server on disk, got %d", len(store.Servers))
	}

	if store.LastSynced.IsZero() {
		t.Fatal("expected last_synced to be set")
	}
}

func TestLoadRestoresFromDisk(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "servers.json")
	store := CacheStore{
		LastSynced: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Servers:    []ServerResponse{sampleServer("ns/cached", "From disk")},
	}

	data, _ := json.Marshal(store)
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		t.Fatalf("failed to seed cache: %v", err)
	}

	cache := NewCacheWithPath(nil, cachePath)
	if err := cache.Load(); err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if cache.Count() != 1 {
		t.Fatalf("expected 1 server, got %d", cache.Count())
	}

	if cache.LastSynced().IsZero() {
		t.Fatal("expected last_synced to be restored")
	}
}

func TestLoadReturnsEmptyWhenFileMissing(t *testing.T) {
	cache := NewCacheWithPath(nil, filepath.Join(t.TempDir(), "servers.json"))
	if err := cache.Load(); err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if cache.Count() != 0 {
		t.Fatalf("expected 0 servers, got %d", cache.Count())
	}

	if !cache.LastSynced().IsZero() {
		t.Fatal("expected zero last_synced")
	}
}

func TestLoadReturnsEmptyOnCorruptedFile(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "servers.json")
	if err := os.WriteFile(cachePath, []byte("{bad json"), 0o644); err != nil {
		t.Fatalf("failed to write corrupt cache: %v", err)
	}

	cache := NewCacheWithPath(nil, cachePath)
	if err := cache.Load(); err != nil {
		t.Fatalf("expected load to succeed on corrupt file: %v", err)
	}

	if cache.Count() != 0 {
		t.Fatalf("expected 0 servers on corrupt file, got %d", cache.Count())
	}
}

func TestSearchMatchesName(t *testing.T) {
	cache := NewCacheWithPath(nil, filepath.Join(t.TempDir(), "servers.json"))
	cache.store.Servers = []ServerResponse{
		sampleServer("io.github.user/sentry-mcp", "Error tracking"),
		sampleServer("io.github.user/jira-mcp", "Project management"),
	}

	results := cache.Search("sentry")
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}

	if results[0].Server.Name != "io.github.user/sentry-mcp" {
		t.Fatalf("unexpected match: %s", results[0].Server.Name)
	}
}

func TestSearchMatchesDescription(t *testing.T) {
	cache := NewCacheWithPath(nil, filepath.Join(t.TempDir(), "servers.json"))
	cache.store.Servers = []ServerResponse{
		sampleServer("ns/server-a", "Error tracking tool"),
		sampleServer("ns/server-b", "File management"),
	}

	results := cache.Search("error")
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}
}

func TestSearchMatchesTitle(t *testing.T) {
	cache := NewCacheWithPath(nil, filepath.Join(t.TempDir(), "servers.json"))
	cache.store.Servers = []ServerResponse{
		{Server: ServerJSON{Name: "ns/a", Title: "Sentry Integration", Description: "MCP", Version: "1.0.0"}},
		sampleServer("ns/b", "Other"),
	}

	results := cache.Search("sentry")
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}
}

func TestSearchIsCaseInsensitive(t *testing.T) {
	cache := NewCacheWithPath(nil, filepath.Join(t.TempDir(), "servers.json"))
	cache.store.Servers = []ServerResponse{
		sampleServer("ns/Sentry-MCP", "Error Tracking"),
	}

	results := cache.Search("SENTRY")
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}
}

func TestSearchEmptyQueryReturnsAll(t *testing.T) {
	cache := NewCacheWithPath(nil, filepath.Join(t.TempDir(), "servers.json"))
	cache.store.Servers = []ServerResponse{
		sampleServer("ns/a", "A"),
		sampleServer("ns/b", "B"),
	}

	results := cache.Search("")
	if len(results) != 2 {
		t.Fatalf("expected 2 results for empty query, got %d", len(results))
	}
}

func TestSearchNoMatches(t *testing.T) {
	cache := NewCacheWithPath(nil, filepath.Join(t.TempDir(), "servers.json"))
	cache.store.Servers = []ServerResponse{
		sampleServer("ns/server-a", "A server"),
	}

	results := cache.Search("nonexistent")
	if len(results) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(results))
	}
}

func TestAllReturnsCopy(t *testing.T) {
	cache := NewCacheWithPath(nil, filepath.Join(t.TempDir(), "servers.json"))
	cache.store.Servers = []ServerResponse{sampleServer("ns/a", "A")}

	all := cache.All()
	all[0].Server.Name = "mutated"

	if cache.store.Servers[0].Server.Name == "mutated" {
		t.Fatal("All() should return a copy, not a reference")
	}
}

func TestSyncNetworkErrorPreservesStaleCache(t *testing.T) {
	mock := &mockLister{err: errors.New("network timeout")}

	cache := NewCacheWithPath(mock, filepath.Join(t.TempDir(), "servers.json"))
	cache.store.LastSynced = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cache.store.Servers = []ServerResponse{sampleServer("ns/stale", "Stale data")}

	err := cache.Sync()
	if err == nil {
		t.Fatal("expected error on network failure")
	}

	if cache.Count() != 1 {
		t.Fatalf("expected stale cache preserved, got %d servers", cache.Count())
	}

	if cache.store.Servers[0].Server.Name != "ns/stale" {
		t.Fatal("expected stale server data preserved")
	}
}

func TestColdSyncNetworkErrorOnEmptyCache(t *testing.T) {
	mock := &mockLister{err: errors.New("connection refused")}

	cache := NewCacheWithPath(mock, filepath.Join(t.TempDir(), "servers.json"))

	err := cache.Sync()
	if err == nil {
		t.Fatal("expected error on network failure")
	}

	if cache.Count() != 0 {
		t.Fatalf("expected empty cache after failed cold sync, got %d", cache.Count())
	}
}

func TestSyncCreatesDirectoryAndFile(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "nested", "dir", "servers.json")
	mock := &mockLister{
		pages: []ServerListResponse{
			{Servers: []ServerResponse{}, Metadata: Metadata{Count: 0}},
		},
	}

	cache := NewCacheWithPath(mock, cachePath)
	if err := cache.Sync(); err != nil {
		t.Fatalf("expected sync to succeed: %v", err)
	}

	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("expected cache file to be created: %v", err)
	}
}

func TestSyncThenLoadRoundTrip(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "servers.json")
	mock := &mockLister{
		pages: []ServerListResponse{
			{
				Servers:  []ServerResponse{sampleServer("ns/roundtrip", "Test")},
				Metadata: Metadata{Count: 1},
			},
		},
	}

	cache1 := NewCacheWithPath(mock, cachePath)
	if err := cache1.Sync(); err != nil {
		t.Fatalf("expected sync to succeed: %v", err)
	}

	cache2 := NewCacheWithPath(nil, cachePath)
	if err := cache2.Load(); err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if cache2.Count() != 1 {
		t.Fatalf("expected 1 server after reload, got %d", cache2.Count())
	}

	if cache2.LastSynced().IsZero() {
		t.Fatal("expected last_synced preserved after reload")
	}
}

func TestNewCacheUsesDefaultPath(t *testing.T) {
	cache := NewCache(nil)
	if cache.path == "" {
		t.Fatal("expected default path to be set")
	}
}
