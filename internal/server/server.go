package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/CalmOli/referro-mcp/internal/referro"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// New builds the referro MCP server with all tools registered. The four tools
// keep the names from the original design (list_designs, get_design,
// get_design_images, get_design_workflow) but are backed by refero.design's
// free public API.
func New(client *referro.Client) *server.MCPServer {
	s := server.NewMCPServer(
		"referro-mcp",
		"0.2.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	s.AddTool(listDesignsTool(), mcp.NewTypedToolHandler(listDesignsHandler(client)))
	s.AddTool(getDesignTool(), mcp.NewTypedToolHandler(getDesignHandler(client)))
	s.AddTool(getDesignImagesTool(), mcp.NewTypedToolHandler(getDesignImagesHandler(client)))
	s.AddTool(getDesignWorkflowTool(), mcp.NewTypedToolHandler(getDesignWorkflowHandler(client)))

	return s
}

// --- list_designs ---

type listDesignsArgs struct {
	Query    string `json:"query,omitempty" jsonschema:"description=Search query (site, app, industry, topic). Empty returns popular results"`
	Platform string `json:"platform,omitempty" jsonschema:"description=Optional: 'web' for websites or 'ios' for iOS apps. Defaults to web"`
	Page     int    `json:"page,omitempty" jsonschema:"description=Page number (1-based). Default 1"`
}

func listDesignsTool() mcp.Tool {
	return mcp.NewTool(
		"list_designs",
		mcp.WithDescription("Search refero.design's catalog of captured designs (websites and iOS apps) by query, industry, or visual direction. Returns screens with their site/app, page URL, colors, fonts, page types, and image URLs."),
		mcp.WithString("query", mcp.Description("Search query (site, app, industry, topic). Empty returns popular results")),
		mcp.WithString("platform", mcp.Description("Optional: 'web' for websites or 'ios' for iOS apps. Defaults to web")),
		mcp.WithNumber("page", mcp.Description("Page number (1-based). Default 1")),
	)
}

func listDesignsHandler(client *referro.Client) func(context.Context, mcp.CallToolRequest, listDesignsArgs) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest, args listDesignsArgs) (*mcp.CallToolResult, error) {
		page := args.Page
		if page < 1 {
			page = 1
		}
		var out any
		var err error
		if args.Platform == "ios" {
			out, err = client.SearchApps(ctx, args.Query, page, 24)
		} else {
			out, err = client.SearchScreens(ctx, args.Query, page, 24)
		}
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}
		return marshalResult(out)
	}
}

// --- get_design ---

type getDesignArgs struct {
	// Ref is a screen UUID, a site/app name or domain, or a numeric screen ID.
	Ref string `json:"ref" jsonschema:"required,description=Screen UUID, site/app name or domain, or numeric screen ID"`
}

func getDesignTool() mcp.Tool {
	return mcp.NewTool(
		"get_design",
		mcp.WithDescription("Fetch a single design (captured screen) by UUID, site/app name or domain, or numeric screen ID. Returns full image URLs, page URL, colors, fonts, page types, design patterns, and associated flow IDs."),
		mcp.WithString("ref", mcp.Required(), mcp.Description("Screen UUID, site/app name or domain, or numeric screen ID")),
	)
}

func getDesignHandler(client *referro.Client) func(context.Context, mcp.CallToolRequest, getDesignArgs) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest, args getDesignArgs) (*mcp.CallToolResult, error) {
		if strings.TrimSpace(args.Ref) == "" {
			return mcp.NewToolResultError("ref is required"), nil
		}
		s, err := resolveScreen(ctx, client, args.Ref)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to fetch design: %v", err)), nil
		}
		return marshalResult(s)
	}
}

// --- get_design_images ---

type getDesignImagesArgs struct {
	Ref string `json:"ref" jsonschema:"required,description=Screen UUID, site/app name or domain, or numeric screen ID"`
}

func getDesignImagesTool() mcp.Tool {
	return mcp.NewTool(
		"get_design_images",
		mcp.WithDescription("Fetch the image URLs for a design (full-resolution, thumbnail, and preview)."),
		mcp.WithString("ref", mcp.Required(), mcp.Description("Screen UUID, site/app name or domain, or numeric screen ID")),
	)
}

func getDesignImagesHandler(client *referro.Client) func(context.Context, mcp.CallToolRequest, getDesignImagesArgs) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest, args getDesignImagesArgs) (*mcp.CallToolResult, error) {
		if strings.TrimSpace(args.Ref) == "" {
			return mcp.NewToolResultError("ref is required"), nil
		}
		s, err := resolveScreen(ctx, client, args.Ref)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to fetch design: %v", err)), nil
		}
		imgs := map[string]any{
			"uuid":            s.UUID,
			"full_resolution": s.URL,
			"thumbnail":       s.ThumbnailURL,
			"preview":         s.PreviewURL,
			"preview_full":    s.PreviewFull,
		}
		return marshalResult(imgs)
	}
}

// --- get_design_workflow ---

type getDesignWorkflowArgs struct {
	// ID is the flow's numeric ID (a flow is a multi-step user journey).
	ID int `json:"id" jsonschema:"required,description=The flow's numeric ID"`
}

func getDesignWorkflowTool() mcp.Tool {
	return mcp.NewTool(
		"get_design_workflow",
		mcp.WithDescription("Fetch a workflow (a flow: a multi-step user journey like checkout, onboarding, or password reset) by its numeric ID. Returns the ordered steps with their screen images."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("The flow's numeric ID")),
	)
}

func getDesignWorkflowHandler(client *referro.Client) func(context.Context, mcp.CallToolRequest, getDesignWorkflowArgs) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest, args getDesignWorkflowArgs) (*mcp.CallToolResult, error) {
		if args.ID < 1 {
			return mcp.NewToolResultError("id is required"), nil
		}
		f, err := client.GetFlow(ctx, args.ID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to fetch workflow: %v", err)), nil
		}
		return marshalResult(f)
	}
}

// --- helpers ---

// resolveScreen turns a user-supplied ref into a Screen. It accepts a UUID, a
// numeric screen ID (resolved via search), or a site/app name or domain
// (resolved via search and matched by name/domain).
func resolveScreen(ctx context.Context, client *referro.Client, ref string) (*referro.Screen, error) {
	ref = strings.TrimSpace(ref)

	// UUID (8-4-4-4-12 hex) — fetch directly.
	if isUUID(ref) {
		return client.GetScreen(ctx, ref)
	}

	// Numeric screen ID — search and match by numeric id.
	if id, err := strconv.Atoi(ref); err == nil {
		res, err := client.SearchScreens(ctx, "", 1, 24)
		if err != nil {
			return nil, err
		}
		for _, s := range res.Records {
			if s.ID == id {
				return &s, nil
			}
		}
		return nil, fmt.Errorf("no screen with id %d", id)
	}

	// Name or domain — search both catalogs and match by site/app name or domain.
	lower := strings.ToLower(ref)
	if res, err := client.SearchScreens(ctx, ref, 1, 24); err == nil {
		for _, s := range res.Records {
			if s.Site != nil {
				if strings.EqualFold(s.Site.Name, ref) || strings.EqualFold(s.Site.Domain, lower) {
					return &s, nil
				}
			}
		}
	}
	if res, err := client.SearchApps(ctx, ref, 1, 24); err == nil {
		for _, s := range res.Records {
			if s.App != nil && strings.EqualFold(s.App.Name, ref) {
				return &s, nil
			}
		}
	}
	return nil, fmt.Errorf("no design matching %q", ref)
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
			continue
		}
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}

func marshalResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("failed to encode result"), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}
