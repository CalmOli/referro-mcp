package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/CalmOli/referro-mcp/internal/referro"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var fixtureBase string

func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	fixturesDir := filepath.Join(filepath.Dir(file), "..", "..", "fixtures")
	srv := httptest.NewServer(http.FileServer(http.Dir(fixturesDir)))
	fixtureBase = srv.URL
	defer srv.Close()
	m.Run()
}

// callTool runs one MCP tool call through the real client -> server stack and
// returns the text payload.
func callTool(t *testing.T, c *client.Client, name string, args map[string]any) string {
	t.Helper()

	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	res, err := c.CallTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	if res.IsError {
		return "<error>"
	}
	// Collect all text content blocks.
	var out string
	for _, block := range res.Content {
		if text, ok := block.(mcp.TextContent); ok {
			out += text.Text
		}
	}
	return out
}

func TestServerEndToEnd(t *testing.T) {
	clientImpl := referro.NewClient(referro.WithBaseURL(fixtureBase))
	srv := New(clientImpl)

	c, err := client.NewInProcessClient(srv)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}
	t.Cleanup(func() { c.Close() })

	// MCP requires the initialize handshake before any other call.
	if _, err := c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo:      mcp.Implementation{Name: "referro-mcp-test", Version: "0.1.0"},
		},
	}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// tools/list
	tools, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]string{}
	for _, tl := range tools.Tools {
		names[tl.Name] = tl.Description
	}
	for _, want := range []string{"list_designs", "get_design", "get_design_images", "get_design_workflow"} {
		if _, ok := names[want]; !ok {
			t.Errorf("expected tool %q to be registered", want)
		}
	}

	// list_designs
	out := callTool(t, c, "list_designs", nil)
	var designs []referro.Design
	if err := json.Unmarshal([]byte(out), &designs); err != nil {
		t.Fatalf("list_designs returned non-JSON: %v\n%s", err, out)
	}
	if len(designs) != 4 {
		t.Fatalf("expected 4 designs, got %d", len(designs))
	}

	// list_designs with type filter
	out = callTool(t, c, "list_designs", map[string]any{"type": "mobile"})
	designs = nil
	if err := json.Unmarshal([]byte(out), &designs); err != nil {
		t.Fatalf("filtered list_designs non-JSON: %v", err)
	}
	if len(designs) != 1 || designs[0].ID != "mobile-app" {
		t.Fatalf("mobile filter: got %+v", designs)
	}

	// get_design fetches markdown
	out = callTool(t, c, "get_design", map[string]any{"ref": "/designs/landing-page/index.html"})
	var d referro.Design
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		t.Fatalf("get_design non-JSON: %v\n%s", err, out)
	}
	if d.MD == "" || !stringsContains(d.MD, "Ship faster with Referro") {
		t.Fatalf("get_design markdown missing spec content: %q...", d.MD)
	}

	// get_design_images
	out = callTool(t, c, "get_design_images", map[string]any{"ref": "/designs/mobile-app/index.html"})
	var imgs []referro.Image
	if err := json.Unmarshal([]byte(out), &imgs); err != nil {
		t.Fatalf("get_design_images non-JSON: %v\n%s", err, out)
	}
	if len(imgs) < 3 {
		t.Fatalf("expected >=3 images, got %d", len(imgs))
	}

	// get_design_workflow
	out = callTool(t, c, "get_design_workflow", map[string]any{"ref": "/designs/landing-page/index.html"})
	var w referro.Workflow
	if err := json.Unmarshal([]byte(out), &w); err != nil {
		t.Fatalf("get_design_workflow non-JSON: %v\n%s", err, out)
	}
	if len(w.Steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(w.Steps))
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}