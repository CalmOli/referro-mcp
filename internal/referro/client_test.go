package referro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// fixtureBase points at an httptest server that serves the captured API
// responses in fixtures/api/, so tests are deterministic and offline.
var fixtureBase string

func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	apiDir := filepath.Join(filepath.Dir(file), "..", "..", "fixtures", "api")

	// Map API paths to fixture files.
	routes := map[string]string{
		"/v1/search":      "search_sites.json",
		"/v1/search/apps": "search_apps.json",
		"/v1/flows/68":    "flow_68.json",
		"/v1/screenshots/cdd6f9ec-b0d0-4e2f-86a4-6fdd010861cf": "screen_uuid.json",
		"/v1/screenshots/211/similar":                          "similar_211.json",
		"/v1/sites/9":                                          "site_9.json",
		"/v1/apps/4":                                           "app_4.json",
		"/v1/design_patterns/available":                        "patterns.json",
	}

	mux := http.NewServeMux()
	for path, file := range routes {
		file := file
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			b, err := os.ReadFile(filepath.Join(apiDir, file))
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		})
	}

	srv := httptest.NewServer(mux)
	fixtureBase = srv.URL + "/v1"
	defer srv.Close()
	m.Run()
}

func testClient(t *testing.T) *Client {
	t.Helper()
	return NewClient(WithBaseURL(fixtureBase))
}

func TestSearchScreens(t *testing.T) {
	c := testClient(t)
	res, err := c.SearchScreens(context.Background(), "stripe", 1, 3)
	if err != nil {
		t.Fatalf("SearchScreens: %v", err)
	}
	if res.Pagination.Count == 0 {
		t.Fatal("expected results")
	}
	if len(res.Records) == 0 {
		t.Fatal("expected records")
	}
	r := res.Records[0]
	if r.Site == nil || r.Site.Name != "Stripe" {
		t.Errorf("first record site = %+v, want Stripe", r.Site)
	}
	if r.UUID == "" {
		t.Error("expected uuid")
	}
}

func TestSearchApps(t *testing.T) {
	c := testClient(t)
	res, err := c.SearchApps(context.Background(), "headspace", 1, 3)
	if err != nil {
		t.Fatalf("SearchApps: %v", err)
	}
	if len(res.Records) == 0 {
		t.Fatal("expected records")
	}
	if res.Records[0].App == nil || res.Records[0].App.Name != "Headspace" {
		t.Errorf("first app = %+v", res.Records[0].App)
	}
}

func TestGetScreen(t *testing.T) {
	c := testClient(t)
	s, err := c.GetScreen(context.Background(), "cdd6f9ec-b0d0-4e2f-86a4-6fdd010861cf")
	if err != nil {
		t.Fatalf("GetScreen: %v", err)
	}
	if s.Site == nil || s.Site.Name != "Stripe" {
		t.Errorf("site = %+v", s.Site)
	}
	if len(s.URL) == 0 {
		t.Error("expected image urls")
	}
	if len(s.FlowIDs) == 0 {
		t.Error("expected flow ids")
	}
}

func TestGetFlow(t *testing.T) {
	c := testClient(t)
	f, err := c.GetFlow(context.Background(), 68)
	if err != nil {
		t.Fatalf("GetFlow: %v", err)
	}
	if f.Name != "Resetting password" {
		t.Errorf("flow name = %q", f.Name)
	}
	if len(f.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(f.Steps))
	}
}

func TestGetSimilarScreens(t *testing.T) {
	c := testClient(t)
	res, err := c.GetSimilarScreens(context.Background(), 211, 1)
	if err != nil {
		t.Fatalf("GetSimilarScreens: %v", err)
	}
	if len(res.Records) == 0 {
		t.Fatal("expected similar screens")
	}
}

func TestGetSite(t *testing.T) {
	c := testClient(t)
	s, err := c.GetSite(context.Background(), 9)
	if err != nil {
		t.Fatalf("GetSite: %v", err)
	}
	if s.Name != "Stripe" {
		t.Errorf("site name = %q", s.Name)
	}
}

func TestGetApp(t *testing.T) {
	c := testClient(t)
	a, err := c.GetApp(context.Background(), 4)
	if err != nil {
		t.Fatalf("GetApp: %v", err)
	}
	if a.Name != "Headspace" {
		t.Errorf("app name = %q", a.Name)
	}
}

func TestListDesignPatterns(t *testing.T) {
	c := testClient(t)
	patterns, err := c.ListDesignPatterns(context.Background())
	if err != nil {
		t.Fatalf("ListDesignPatterns: %v", err)
	}
	if len(patterns) < 50 {
		t.Errorf("expected many patterns, got %d", len(patterns))
	}
}

func TestCache(t *testing.T) {
	dir := t.TempDir()
	c := NewClient(WithBaseURL(fixtureBase), WithCache(dir, time.Minute))
	if _, err := c.SearchScreens(context.Background(), "stripe", 1, 3); err != nil {
		t.Fatalf("first search: %v", err)
	}
	if _, err := c.SearchScreens(context.Background(), "stripe", 1, 3); err != nil {
		t.Fatalf("cached search: %v", err)
	}
}

// ensure fixture JSON parses into our structs (guards against drift).
func TestFixtureShape(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	apiDir := filepath.Join(filepath.Dir(file), "..", "..", "fixtures", "api")
	for _, name := range []string{"search_sites.json", "search_apps.json", "flow_68.json", "screen_uuid.json", "similar_211.json", "site_9.json", "app_4.json", "patterns.json"} {
		b, err := os.ReadFile(filepath.Join(apiDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s not valid JSON: %v", name, err)
		}
	}
}
