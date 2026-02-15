package registry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	ts := httptest.NewServer(handler)
	client := NewClientWithBaseURL(ts.URL)

	return ts, client
}

func TestListServersReturnsResults(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}

		if !strings.HasPrefix(r.URL.Path, "/v0.1/servers") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		if r.URL.Query().Get("version") != "latest" {
			t.Fatal("expected version=latest query parameter")
		}

		resp := ServerListResponse{
			Servers: []ServerResponse{
				{
					Server: ServerJSON{
						Name:        "io.github.user/test-server",
						Description: "A test server",
						Version:     "1.0.0",
					},
				},
			},
			Metadata: Metadata{
				Count: 1,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	result, err := client.ListServers(ListOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(result.Servers))
	}

	if result.Servers[0].Server.Name != "io.github.user/test-server" {
		t.Fatalf("unexpected server name: %s", result.Servers[0].Server.Name)
	}
}

func TestListServersPassesSearchParam(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		search := r.URL.Query().Get("search")
		if search != "sentry" {
			t.Fatalf("expected search=sentry, got %q", search)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServerListResponse{
			Servers:  []ServerResponse{},
			Metadata: Metadata{Count: 0},
		})
	})
	defer ts.Close()

	_, err := client.ListServers(ListOptions{Search: "sentry"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestListServersPassesCursorParam(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		if cursor != "abc123" {
			t.Fatalf("expected cursor=abc123, got %q", cursor)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServerListResponse{
			Servers:  []ServerResponse{},
			Metadata: Metadata{Count: 0},
		})
	})
	defer ts.Close()

	_, err := client.ListServers(ListOptions{Cursor: "abc123"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestListServersPassesUpdatedSinceParam(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		updatedSince := r.URL.Query().Get("updated_since")
		if updatedSince != "2025-01-01T00:00:00Z" {
			t.Fatalf("expected updated_since timestamp, got %q", updatedSince)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServerListResponse{
			Servers:  []ServerResponse{},
			Metadata: Metadata{Count: 0},
		})
	})
	defer ts.Close()

	_, err := client.ListServers(ListOptions{UpdatedSince: "2025-01-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestListServersClipsLimitToMax(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		limit := r.URL.Query().Get("limit")
		if limit != "100" {
			t.Fatalf("expected limit=100, got %q", limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServerListResponse{
			Servers:  []ServerResponse{},
			Metadata: Metadata{Count: 0},
		})
	})
	defer ts.Close()

	_, err := client.ListServers(ListOptions{Limit: 500})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestListServersUsesDefaultLimit(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		limit := r.URL.Query().Get("limit")
		if limit != "30" {
			t.Fatalf("expected default limit=30, got %q", limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServerListResponse{
			Servers:  []ServerResponse{},
			Metadata: Metadata{Count: 0},
		})
	})
	defer ts.Close()

	_, err := client.ListServers(ListOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestListServersPaginationCursor(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		resp := ServerListResponse{
			Servers: []ServerResponse{
				{Server: ServerJSON{Name: "io.github.user/server", Description: "test", Version: "1.0.0"}},
			},
			Metadata: Metadata{
				Count:      1,
				NextCursor: "next-page-cursor",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	result, err := client.ListServers(ListOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Metadata.NextCursor != "next-page-cursor" {
		t.Fatalf("expected next cursor, got %q", result.Metadata.NextCursor)
	}
}

func TestGetServerLatestReturnsDetails(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}

		expectedPath := "/v0.1/servers/io.github.user%2Ftest-server/versions/latest"
		if r.URL.RawPath != expectedPath && r.URL.Path != "/v0.1/servers/io.github.user%2Ftest-server/versions/latest" {
			// Check the raw path contains encoded slash
			if !strings.Contains(r.URL.RawPath, "%2F") && !strings.Contains(r.RequestURI, "%2F") {
				t.Fatalf("expected URL-encoded slash in path, got path=%q rawPath=%q", r.URL.Path, r.URL.RawPath)
			}
		}

		resp := ServerResponse{
			Server: ServerJSON{
				Name:        "io.github.user/test-server",
				Description: "A test MCP server",
				Version:     "2.0.0",
				Remotes: []Transport{
					{Type: "sse", URL: "https://mcp.example.com/sse"},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	result, err := client.GetServerLatest("io.github.user/test-server")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Server.Name != "io.github.user/test-server" {
		t.Fatalf("unexpected name: %s", result.Server.Name)
	}

	if result.Server.Version != "2.0.0" {
		t.Fatalf("unexpected version: %s", result.Server.Version)
	}

	if len(result.Server.Remotes) != 1 {
		t.Fatalf("expected 1 remote, got %d", len(result.Server.Remotes))
	}

	if result.Server.Remotes[0].URL != "https://mcp.example.com/sse" {
		t.Fatalf("unexpected remote URL: %s", result.Server.Remotes[0].URL)
	}
}

func TestGetServerLatestRejectsEmptyName(t *testing.T) {
	client := NewClient()

	_, err := client.GetServerLatest("")
	if err == nil {
		t.Fatal("expected error for empty server name")
	}
}

func TestGetServerLatestRejectsWhitespaceName(t *testing.T) {
	client := NewClient()

	_, err := client.GetServerLatest("   ")
	if err == nil {
		t.Fatal("expected error for whitespace server name")
	}
}

func TestAPIErrorParsing(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)

		apiErr := APIError{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: 404,
			Detail: "Server io.github.user/nonexistent not found",
		}

		json.NewEncoder(w).Encode(apiErr)
	})
	defer ts.Close()

	_, err := client.GetServerLatest("io.github.user/nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}

	if apiErr.Status != 404 {
		t.Fatalf("expected status 404, got %d", apiErr.Status)
	}

	if !strings.Contains(apiErr.Error(), "not found") {
		t.Fatalf("expected error message to contain 'not found', got %q", apiErr.Error())
	}
}

func TestAPIErrorFallbackWhenBodyNotJSON(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	})
	defer ts.Close()

	_, err := client.ListServers(ListOptions{})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "500") {
		t.Fatalf("expected error to mention status code, got %q", errMsg)
	}

	if !strings.Contains(errMsg, "internal error") {
		t.Fatalf("expected error to include body snippet, got %q", errMsg)
	}
}

func TestAPIErrorFallbackWhenBodyEmpty(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	defer ts.Close()

	_, err := client.ListServers(ListOptions{})
	if err == nil {
		t.Fatal("expected error for 502 response")
	}

	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected error to mention status code, got %q", err.Error())
	}

	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("expected error to mention empty response, got %q", err.Error())
	}
}

func TestAPIErrorWithValidationErrors(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusBadRequest)

		apiErr := APIError{
			Title:  "Bad Request",
			Status: 400,
			Detail: "Validation failed",
			Errors: []ErrorDetail{
				{Location: "query.limit", Message: "must be positive"},
			},
		}

		json.NewEncoder(w).Encode(apiErr)
	})
	defer ts.Close()

	_, err := client.ListServers(ListOptions{})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if len(apiErr.Errors) != 1 {
		t.Fatalf("expected 1 error detail, got %d", len(apiErr.Errors))
	}
}

func TestClientSetsUserAgentHeader(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if !strings.HasPrefix(ua, "mcp-wire/") {
			t.Fatalf("expected User-Agent to start with mcp-wire/, got %q", ua)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServerListResponse{
			Servers:  []ServerResponse{},
			Metadata: Metadata{Count: 0},
		})
	})
	defer ts.Close()

	_, err := client.ListServers(ListOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestClientSetsAcceptHeader(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if accept != "application/json" {
			t.Fatalf("expected Accept: application/json, got %q", accept)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServerListResponse{
			Servers:  []ServerResponse{},
			Metadata: Metadata{Count: 0},
		})
	})
	defer ts.Close()

	_, err := client.ListServers(ListOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNewClientUsesDefaultBaseURL(t *testing.T) {
	client := NewClient()
	if client.baseURL != DefaultBaseURL {
		t.Fatalf("expected default base URL %q, got %q", DefaultBaseURL, client.baseURL)
	}
}

func TestNewClientWithBaseURLStripsTrailingSlash(t *testing.T) {
	client := NewClientWithBaseURL("https://example.com/")
	if client.baseURL != "https://example.com" {
		t.Fatalf("expected trailing slash stripped, got %q", client.baseURL)
	}
}

func TestGetServerLatestParsesPackages(t *testing.T) {
	ts, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		resp := ServerResponse{
			Server: ServerJSON{
				Name:        "io.github.user/npm-server",
				Description: "An npm-based server",
				Version:     "1.0.0",
				Packages: []Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/mcp-server",
						Transport:    Transport{Type: "stdio"},
						RuntimeHint:  "npx",
						EnvironmentVariables: []KeyValueInput{
							{Name: "API_KEY", Description: "API key", IsRequired: true, IsSecret: true},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	result, err := client.GetServerLatest("io.github.user/npm-server")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Server.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(result.Server.Packages))
	}

	pkg := result.Server.Packages[0]
	if pkg.RegistryType != "npm" {
		t.Fatalf("expected npm registry type, got %q", pkg.RegistryType)
	}

	if pkg.RuntimeHint != "npx" {
		t.Fatalf("expected npx runtime hint, got %q", pkg.RuntimeHint)
	}

	if len(pkg.EnvironmentVariables) != 1 {
		t.Fatalf("expected 1 env var, got %d", len(pkg.EnvironmentVariables))
	}

	if !pkg.EnvironmentVariables[0].IsSecret {
		t.Fatal("expected API_KEY to be marked as secret")
	}
}

func TestAPIErrorErrorMethod(t *testing.T) {
	tests := []struct {
		name     string
		err      APIError
		expected string
	}{
		{
			name:     "uses detail when present",
			err:      APIError{Detail: "specific detail", Title: "General Title"},
			expected: "specific detail",
		},
		{
			name:     "falls back to title",
			err:      APIError{Title: "Not Found"},
			expected: "Not Found",
		},
		{
			name:     "falls back to generic message",
			err:      APIError{},
			expected: "registry API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestListServersConnectionRefused(t *testing.T) {
	client := NewClientWithBaseURL("http://127.0.0.1:1")

	_, err := client.ListServers(ListOptions{})
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}
