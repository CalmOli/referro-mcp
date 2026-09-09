package referro

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// fixtureBase points at a live httptest server serving the fixtures/ directory,
// so the scraper is exercised end-to-end without any external dependency.
var fixtureBase string

func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	fixturesDir := filepath.Join(filepath.Dir(file), "..", "..", "fixtures")
	srv := httptest.NewServer(http.FileServer(http.Dir(fixturesDir)))
	fixtureBase = srv.URL
	defer srv.Close()
	m.Run()
}

func testClient(t *testing.T) *Client {
	t.Helper()
	return NewClient(WithBaseURL(fixtureBase))
}

func TestListDesigns(t *testing.T) {
	c := testClient(t)
	designs, err := c.ListDesigns(context.Background())
	if err != nil {
		t.Fatalf("ListDesigns: %v", err)
	}
	if len(designs) < 4 {
		t.Fatalf("expected >=4 designs, got %d", len(designs))
	}

	byID := map[string]Design{}
	for _, d := range designs {
		byID[d.ID] = d
	}

	for _, id := range []string{"landing-page", "mobile-app", "checkout-ui", "dashboard-web"} {
		if _, ok := byID[id]; !ok {
			t.Errorf("expected design %q in list", id)
		}
	}

	if byID["mobile-app"].Type != TypeMobile {
		t.Errorf("mobile-app type = %q, want mobile", byID["mobile-app"].Type)
	}
	if byID["checkout-ui"].Type != TypeUI {
		t.Errorf("checkout-ui type = %q, want ui", byID["checkout-ui"].Type)
	}
	if byID["landing-page"].Type != TypeWebsite {
		t.Errorf("landing-page type = %q, want website", byID["landing-page"].Type)
	}
}

func TestGetDesignByID(t *testing.T) {
	c := testClient(t)
	d, err := c.GetDesign(context.Background(), "landing-page")
	if err != nil {
		t.Fatalf("GetDesign by id: %v", err)
	}
	if d.ID != "landing-page" {
		t.Errorf("id = %q, want landing-page", d.ID)
	}
	if d.MD == "" || !contains(d.MD, "Ship faster with Referro") {
		t.Errorf("markdown not populated from ID resolution")
	}

	// Unknown ID should be a clean error.
	if _, err := c.GetDesign(context.Background(), "no-such-design"); err == nil {
		t.Error("expected error for unknown design id")
	}
}

func TestGetDesignMarkdown(t *testing.T) {
	c := testClient(t)
	d, err := c.GetDesign(context.Background(), "/designs/landing-page/index.html")
	if err != nil {
		t.Fatalf("GetDesign: %v", err)
	}
	if d.MD == "" {
		t.Fatal("expected design markdown to be populated")
	}
	// The linked spec.md should be fetched.
	if !contains(d.MD, "Ship faster with Referro") {
		t.Errorf("markdown missing linked spec content:\n%s", d.MD)
	}
}

func TestGetDesignImages(t *testing.T) {
	c := testClient(t)
	imgs, err := c.GetImages(context.Background(), "/designs/mobile-app/index.html")
	if err != nil {
		t.Fatalf("GetImages: %v", err)
	}
	if len(imgs) < 3 {
		t.Fatalf("expected >=3 images, got %d", len(imgs))
	}
	// og:image should be first (hero).
	if imgs[0].URL != fixtureBase+"/assets/mobile-home.png" {
		t.Errorf("first image = %q, want hero og:image", imgs[0].URL)
	}
}

func TestGetDesignWorkflow(t *testing.T) {
	c := testClient(t)
	w, err := c.GetWorkflow(context.Background(), "/designs/landing-page/index.html")
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if len(w.Steps) != 5 {
		t.Fatalf("expected 5 workflow steps, got %d", len(w.Steps))
	}
	if w.Steps[0].Title == "" {
		t.Error("first step has empty title")
	}
}

func TestCache(t *testing.T) {
	dir := t.TempDir()
	c := NewClient(WithBaseURL(fixtureBase), WithCache(dir, time.Minute))
	if _, err := c.ListDesigns(context.Background()); err != nil {
		t.Fatalf("first ListDesigns: %v", err)
	}
	// Second call should hit the cache (no error, still works).
	if _, err := c.ListDesigns(context.Background()); err != nil {
		t.Fatalf("cached ListDesigns: %v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
