package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andreagrandi/mcp-wire/internal/app"
)

const (
	DefaultBaseURL = "https://registry.modelcontextprotocol.io"
	apiVersion     = "v0.1"
	maxLimit       = 100
	defaultTimeout = 15 * time.Second
)

// Client is a read-only client for the Official MCP Registry API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a registry client with the default base URL.
func NewClient() *Client {
	return NewClientWithBaseURL(DefaultBaseURL)
}

// NewClientWithBaseURL creates a registry client with a custom base URL.
func NewClientWithBaseURL(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// ListOptions configures a ListServers request.
type ListOptions struct {
	Limit        int
	Cursor       string
	Search       string
	UpdatedSince string
}

// ListServers returns a paginated list of latest-version servers.
func (c *Client) ListServers(opts ListOptions) (*ServerListResponse, error) {
	params := url.Values{}
	params.Set("version", "latest")

	limit := opts.Limit
	if limit <= 0 {
		limit = 30
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	params.Set("limit", fmt.Sprintf("%d", limit))

	if opts.Cursor != "" {
		params.Set("cursor", opts.Cursor)
	}

	if opts.Search != "" {
		params.Set("search", opts.Search)
	}

	if opts.UpdatedSince != "" {
		params.Set("updated_since", opts.UpdatedSince)
	}

	endpoint := fmt.Sprintf("%s/%s/servers?%s", c.baseURL, apiVersion, params.Encode())

	var result ServerListResponse
	if err := c.doGet(endpoint, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetServerLatest returns the latest version details for a server.
//
// The serverName must be in reverse-DNS format (e.g. "io.github.user/server").
// The slash is URL-encoded automatically.
func (c *Client) GetServerLatest(serverName string) (*ServerResponse, error) {
	trimmed := strings.TrimSpace(serverName)
	if trimmed == "" {
		return nil, fmt.Errorf("server name is required")
	}

	encoded := url.PathEscape(trimmed)
	endpoint := fmt.Sprintf("%s/%s/servers/%s/versions/latest", c.baseURL, apiVersion, encoded)

	var result ServerResponse
	if err := c.doGet(endpoint, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) doGet(endpoint string, target any) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("mcp-wire/%s", app.Version))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("registry request failed: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	return nil
}

func parseAPIError(statusCode int, body []byte) error {
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		snippet = strings.TrimSpace(snippet)
		if snippet == "" {
			return fmt.Errorf("registry returned HTTP %d (empty response)", statusCode)
		}

		return fmt.Errorf("registry returned HTTP %d: %s", statusCode, snippet)
	}

	if apiErr.Status == 0 {
		apiErr.Status = statusCode
	}

	return &apiErr
}
