package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
	apiDir := filepath.Join(filepath.Dir(file), "..", "..", "fixtures", "api")

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

	if _, err := c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo:      mcp.Implementation{Name: "referro-mcp-test", Version: "0.2.0"},
		},
	}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// tools/list
	tools, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tl := range tools.Tools {
		names[tl.Name] = true
	}
	for _, want := range []string{"list_designs", "get_design", "get_design_images", "get_design_workflow"} {
		if !names[want] {
			t.Errorf("expected tool %q to be registered", want)
		}
	}

	// list_designs
	out := callTool(t, c, "list_designs", map[string]any{"query": "stripe"})
	var search referro.ScreenSearchResponse
	if err := json.Unmarshal([]byte(out), &search); err != nil {
		t.Fatalf("list_designs non-JSON: %v\n%s", err, out)
	}
	if len(search.Records) == 0 {
		t.Fatal("list_designs returned no records")
	}
	uuid := search.Records[0].UUID
	fid := 0
	if len(search.Records[0].FlowIDs) > 0 {
		fid = search.Records[0].FlowIDs[0]
	}

	// get_design by uuid
	out = callTool(t, c, "get_design", map[string]any{"ref": uuid})
	var screen referro.Screen
	if err := json.Unmarshal([]byte(out), &screen); err != nil {
		t.Fatalf("get_design non-JSON: %v", err)
	}
	if screen.UUID != uuid {
		t.Errorf("get_design uuid mismatch")
	}

	// get_design_images
	out = callTool(t, c, "get_design_images", map[string]any{"ref": uuid})
	var imgs map[string]any
	if err := json.Unmarshal([]byte(out), &imgs); err != nil {
		t.Fatalf("get_design_images non-JSON: %v", err)
	}
	if imgs["full_resolution"] == nil {
		t.Error("get_design_images missing full_resolution")
	}

	// get_design_workflow
	if fid > 0 {
		out = callTool(t, c, "get_design_workflow", map[string]any{"id": fid})
		var flow referro.Flow
		if err := json.Unmarshal([]byte(out), &flow); err != nil {
			t.Fatalf("get_design_workflow non-JSON: %v", err)
		}
		if len(flow.Steps) == 0 {
			t.Error("workflow returned no steps")
		}
	}
}
