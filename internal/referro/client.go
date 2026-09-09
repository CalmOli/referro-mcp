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

// DefaultBaseURL is the site the client scrapes unless REFERRO_BASE_URL says otherwise.
const DefaultBaseURL = "https://referro.design"

// Client scrapes referro.design. It is a scraper, not an API client: it fetches
// HTML (or raw markdown) and extracts designs, images, and workflows from the
// page structure, so it survives referro.design having no public API.
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
// fetched pages for ttl. Caching matters here: coding agents re-invoke tools
// often, and we should not re-scrape the site on every call.
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

// Resolve turns a possibly-relative path into an absolute URL on the site.
func (c *Client) Resolve(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.BaseURL + "/" + strings.TrimLeft(path, "/")
}

type fetched struct {
	body     []byte
	url      string
	ctype    string
	fetchedAt time.Time
}

// fetch returns the page at path (absolute or relative), using the disk cache
// when it is enabled and fresh.
func (c *Client) fetch(ctx context.Context, path string) (*fetched, error) {
	u := c.Resolve(path)

	if c.CacheDir != "" {
		if f, ok := c.cacheGet(u); ok && time.Since(f.fetchedAt) < c.cacheTTL {
			return f, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	// A scraper should present as a browser, not as a generic client.
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36 referro-mcp/0.1")
	req.Header.Set("Accept", "text/html,text/markdown,application/xhtml+xml,application/json;q=0.9,*/*;q=0.8")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", u, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", u, err)
	}

	f := &fetched{body: body, url: u, ctype: resp.Header.Get("Content-Type"), fetchedAt: time.Now()}
	if c.CacheDir != "" {
		c.cacheSet(u, f)
	}
	return f, nil
}

// --- disk cache (simple JSON files keyed by hashed URL) ---

func (c *Client) cachePath(u string) string {
	h := fmt.Sprintf("%x", []byte(u))[:24]
	return filepath.Join(c.CacheDir, h+".json")
}

func (c *Client) cacheGet(u string) (*fetched, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	b, err := os.ReadFile(c.cachePath(u))
	if err != nil {
		return nil, false
	}
	var f fetched
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, false
	}
	return &f, true
}

func (c *Client) cacheSet(u string, f *fetched) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.MkdirAll(c.CacheDir, 0o755); err != nil {
		return
	}
	b, err := json.Marshal(f)
	if err != nil {
		return
	}
	_ = os.WriteFile(c.cachePath(u), b, 0o644)
}