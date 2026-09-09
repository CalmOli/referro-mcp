package referro

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL is the public refero.design API. It needs no auth.
const DefaultBaseURL = "https://api.refero.design/v1"

// Client talks to refero.design's public JSON API. Unlike the official Refero
// MCP (which requires a paid plan), this client uses the free public API.
type Client struct {
	BaseURL  string
	HTTP     *http.Client
	CacheDir string // empty disables the disk cache
	cacheTTL time.Duration

	mu sync.Mutex
}

type ClientOption func(*Client)

func WithBaseURL(base string) ClientOption {
	return func(c *Client) { c.BaseURL = strings.TrimRight(base, "/") }
}

func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.HTTP = hc }
}

// WithCache enables a disk cache under dir (created on demand) that keeps
// fetched pages for ttl. Coding agents re-invoke tools often, so caching
// avoids hammering the API on every call.
func WithCache(dir string, ttl time.Duration) ClientOption {
	return func(c *Client) {
		c.CacheDir = dir
		c.cacheTTL = ttl
	}
}

func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		BaseURL:  DefaultBaseURL,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
		cacheTTL: 10 * time.Minute,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// getJSON GETs a path on the API and decodes the JSON response into out.
func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	body, err := c.get(ctx, path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	u := c.BaseURL + "/" + strings.TrimLeft(path, "/")

	if c.CacheDir != "" {
		if b, ok := c.cacheGet(u); ok {
			return b, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36 referro-mcp/0.2")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", u, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", u, err)
	}
	if c.CacheDir != "" {
		c.cacheSet(u, body)
	}
	return body, nil
}

// --- API methods ---

// SearchScreens searches the website screenshot catalog.
// q is the search query; page is 1-based; perPage caps results (default 24).
func (c *Client) SearchScreens(ctx context.Context, q string, page, perPage int) (*ScreenSearchResponse, error) {
	path := fmt.Sprintf("search?q=%s&page=%d&per_page=%d", urlEncode(q), page, perPage)
	var out ScreenSearchResponse
	if err := c.getJSON(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchApps searches the iOS app screenshot catalog.
func (c *Client) SearchApps(ctx context.Context, q string, page, perPage int) (*AppSearchResponse, error) {
	path := fmt.Sprintf("search/apps?q=%s&page=%d&per_page=%d", urlEncode(q), page, perPage)
	var out AppSearchResponse
	if err := c.getJSON(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetScreen fetches a single screen by its UUID.
func (c *Client) GetScreen(ctx context.Context, uuid string) (*Screen, error) {
	var out Screen
	if err := c.getJSON(ctx, "screenshots/"+uuid, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetSimilarScreens returns screens visually similar to the given screen ID.
func (c *Client) GetSimilarScreens(ctx context.Context, id int, page int) (*ScreenSearchResponse, error) {
	path := fmt.Sprintf("screenshots/%d/similar?page=%d", id, page)
	var out ScreenSearchResponse
	if err := c.getJSON(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetFlow fetches a flow (workflow) by its numeric ID.
func (c *Client) GetFlow(ctx context.Context, id int) (*Flow, error) {
	var out Flow
	if err := c.getJSON(ctx, fmt.Sprintf("flows/%d", id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetSite fetches a site's metadata by ID.
func (c *Client) GetSite(ctx context.Context, id int) (*Site, error) {
	var out Site
	if err := c.getJSON(ctx, fmt.Sprintf("sites/%d", id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetApp fetches an app's metadata by ID.
func (c *Client) GetApp(ctx context.Context, id int) (*App, error) {
	var out App
	if err := c.getJSON(ctx, fmt.Sprintf("apps/%d", id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListDesignPatterns returns the available design patterns.
func (c *Client) ListDesignPatterns(ctx context.Context) ([]DesignPattern, error) {
	var out []DesignPattern
	if err := c.getJSON(ctx, "design_patterns/available", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- disk cache (JSON files keyed by hashed URL) ---

func (c *Client) cachePath(u string) string {
	h := fmt.Sprintf("%x", []byte(u))[:24]
	return filepath.Join(c.CacheDir, h+".json")
}

func (c *Client) cacheGet(u string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, err := os.ReadFile(c.cachePath(u))
	if err != nil {
		return nil, false
	}
	return b, true
}

func (c *Client) cacheSet(u string, b []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := os.MkdirAll(c.CacheDir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(c.cachePath(u), b, 0o644)
}

func urlEncode(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, " ", "%20"), "&", "%26")
}
