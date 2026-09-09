// referro-mcp is a local MCP server that scrapes referro.design and exposes
// its design artifacts (markdown specs, images, workflows) as MCP tools for
// coding agents. It speaks MCP over stdio: run it as a subprocess from any
// MCP client (OpenCode, KiloCode, Claude Desktop, mcp-inspector, ...).
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/CalmOli/referro-mcp/internal/referro"
	"github.com/CalmOli/referro-mcp/internal/server"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	var (
		baseURL  = flag.String("base-url", envOr("REFERRO_BASE_URL", referro.DefaultBaseURL), "Base URL of the site to scrape (default https://referro.design)")
		cacheDir = flag.String("cache-dir", envOr("REFERRO_CACHE_DIR", ""), "Directory for the page cache (default: off)")
		cacheTTL = flag.Duration("cache-ttl", 10*time.Minute, "How long cached pages stay fresh")
		version  = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *version {
		fmt.Println("referro-mcp 0.1.0")
		return
	}

	client := referro.NewClient(
		referro.WithBaseURL(*baseURL),
	)
	if *cacheDir != "" {
		client = referro.NewClient(
			referro.WithBaseURL(*baseURL),
			referro.WithCache(*cacheDir, *cacheTTL),
		)
	}

	s := server.New(client)

	if err := mcpserver.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "referro-mcp: %v\n", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}