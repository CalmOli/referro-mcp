package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/CalmOli/referro-mcp/internal/referro"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// New builds the referro MCP server with all tools registered.
func New(client *referro.Client) *server.MCPServer {
	s := server.NewMCPServer(
		"referro-mcp",
		"0.1.0",
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
	// Type optionally filters results to a single design type.
	Type string `json:"type,omitempty" jsonschema:"description=Optional filter: website, mobile, or ui"`
}

func listDesignsTool() mcp.Tool {
	return mcp.NewTool(
		"list_designs",
		mcp.WithDescription("List design artifacts available on referro.design (websites, mobile apps, UIs). Optionally filter by type."),
		mcp.WithString("type", mcp.Description("Optional filter: website, mobile, or ui")),
	)
}

func listDesignsHandler(client *referro.Client) func(context.Context, mcp.CallToolRequest, listDesignsArgs) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest, args listDesignsArgs) (*mcp.CallToolResult, error) {
		designs, err := client.ListDesigns(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list designs: %v", err)), nil
		}

		var filter referro.DesignType
		if args.Type != "" {
			filter = referro.DesignType(args.Type)
		}

		out := make([]referro.Design, 0, len(designs))
		for _, d := range designs {
			if filter != "" && d.Type != filter {
				continue
			}
			out = append(out, d)
		}

		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to encode result"), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	}
}

// --- get_design ---

type getDesignArgs struct {
	// Ref is the design ID or a URL/path to the design page.
	Ref string `json:"ref" jsonschema:"required,description=Design ID or URL/path of the design page to fetch"`
}

func getDesignTool() mcp.Tool {
	return mcp.NewTool(
		"get_design",
		mcp.WithDescription("Fetch a design's full details from referro.design: metadata, description, and its markdown spec."),
		mcp.WithString("ref", mcp.Required(), mcp.Description("Design ID or URL/path of the design page to fetch")),
	)
}

func getDesignHandler(client *referro.Client) func(context.Context, mcp.CallToolRequest, getDesignArgs) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest, args getDesignArgs) (*mcp.CallToolResult, error) {
		if args.Ref == "" {
			return mcp.NewToolResultError("ref is required"), nil
		}
		d, err := client.GetDesign(ctx, args.Ref)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to fetch design: %v", err)), nil
		}
		b, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to encode result"), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	}
}

// --- get_design_images ---

type getDesignImagesArgs struct {
	Ref string `json:"ref" jsonschema:"required,description=Design ID or URL/path of the design page to fetch images from"`
}

func getDesignImagesTool() mcp.Tool {
	return mcp.NewTool(
		"get_design_images",
		mcp.WithDescription("Fetch the image assets (mockups, screenshots, assets) for a design on referro.design."),
		mcp.WithString("ref", mcp.Required(), mcp.Description("Design ID or URL/path of the design page")),
	)
}

func getDesignImagesHandler(client *referro.Client) func(context.Context, mcp.CallToolRequest, getDesignImagesArgs) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest, args getDesignImagesArgs) (*mcp.CallToolResult, error) {
		if args.Ref == "" {
			return mcp.NewToolResultError("ref is required"), nil
		}
		imgs, err := client.GetImages(ctx, args.Ref)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to fetch images: %v", err)), nil
		}
		b, err := json.MarshalIndent(imgs, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to encode result"), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	}
}

// --- get_design_workflow ---

type getDesignWorkflowArgs struct {
	Ref string `json:"ref" jsonschema:"required,description=Design ID or URL/path of the design page to fetch the workflow from"`
}

func getDesignWorkflowTool() mcp.Tool {
	return mcp.NewTool(
		"get_design_workflow",
		mcp.WithDescription("Fetch the workflow (ordered steps) that describe how a design on referro.design is produced."),
		mcp.WithString("ref", mcp.Required(), mcp.Description("Design ID or URL/path of the design page")),
	)
}

func getDesignWorkflowHandler(client *referro.Client) func(context.Context, mcp.CallToolRequest, getDesignWorkflowArgs) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest, args getDesignWorkflowArgs) (*mcp.CallToolResult, error) {
		if args.Ref == "" {
			return mcp.NewToolResultError("ref is required"), nil
		}
		w, err := client.GetWorkflow(ctx, args.Ref)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to fetch workflow: %v", err)), nil
		}
		b, err := json.MarshalIndent(w, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to encode result"), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	}
}
