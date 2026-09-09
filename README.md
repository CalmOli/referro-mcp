# referro-mcp

A local [MCP](https://modelcontextprotocol.io) server that scrapes
[referro.design](https://referro.design) and exposes its design artifacts as
tools for coding agents.

referro.design has no public API, so this server is a **scraper**: it fetches
the site's pages and extracts design specs (markdown), image assets, and
workflows from the HTML. It runs as a local CLI speaking MCP over stdio, so any
MCP client can use it — including coding agents like **OpenCode** and
**KiloCode**.

## What it does

The server registers four tools:

| Tool | Description |
|------|-------------|
| `list_designs` | List design artifacts (websites, mobile apps, UIs), optionally filtered by type |
| `get_design` | Fetch a design's metadata, description, and markdown spec |
| `get_design_images` | Fetch the image assets (mockups, screenshots) for a design |
| `get_design_workflow` | Fetch the ordered workflow steps for a design |

`ref` arguments accept either a full URL/path or a bare design ID (e.g.
`landing-page`) as returned by `list_designs`.

## Build

Requires Go 1.25+.

```sh
go build -o referro-mcp ./cmd/referro-mcp
```

## Run

```sh
./referro-mcp
```

The server reads its config from flags or environment variables:

| Flag | Env | Default | Purpose |
|------|-----|---------|---------|
| `-base-url` | `REFERRO_BASE_URL` | `https://referro.design` | Site to scrape |
| `-cache-dir` | `REFERRO_CACHE_DIR` | *(off)* | Disk cache directory |
| `-cache-ttl` | — | `10m` | Cache freshness window |

The disk cache is worth enabling for agent use: coding agents re-invoke tools
often, and the cache stops every call from re-scraping the site.

```sh
./referro-mcp -base-url https://referro.design -cache-dir ~/.cache/referro-mcp
```

## Test

The test suite serves a local fixture site (in `fixtures/`) that mimics
referro.design's structure, so tests run without touching the live domain.

```sh
go test ./...
```

## Wire it into a coding agent

### OpenCode

```sh
opencode mcp add referro --env REFERRO_BASE_URL=https://referro.design -- ./referro-mcp
```

This writes an entry to `~/.config/opencode/opencode.jsonc`:

```jsonc
{
  "mcp": {
    "referro": {
      "type": "local",
      "command": ["/absolute/path/to/referro-mcp"],
      "environment": { "REFERRO_BASE_URL": "https://referro.design" }
    }
  }
}
```

Verify with `opencode mcp list` (should show `referro connected`).

### KiloCode

KiloCode is an OpenCode fork with the same CLI, so the config is identical
under `~/.config/kilo/kilo.jsonc`:

```sh
kilocode mcp add referro --env REFERRO_BASE_URL=https://referro.design -- ./referro-mcp
```

### Any other MCP client

Point it at the binary as a stdio command with `REFERRO_BASE_URL` set. For
example, Claude Desktop:

```json
{
  "mcpServers": {
    "referro": {
      "command": "/absolute/path/to/referro-mcp",
      "env": { "REFERRO_BASE_URL": "https://referro.design" }
    }
  }
}
```

## Scraping notes

The scraper is deliberately tolerant because referro.design has no documented
structure. The heuristics live in `internal/referro/scraper.go`:

- **Index**: any link on the base page whose URL looks like a design
  (`/designs/`, `/specs/`, `.md`, `/workflows/`) becomes a design.
- **Type**: inferred from the URL/title (`website`, `mobile`, `ui`).
- **Markdown**: a linked `.md` file is fetched if present; otherwise the page's
  main content is rendered back to loose markdown.
- **Images**: all `<img>` plus `og:image` (hero first).
- **Workflow**: an `<ol>` if present, else headings that read like steps.

If the live site's structure differs, adjust the selectors in `scraper.go` and
extend the fixture in `fixtures/` to cover it.

## Layout

```
cmd/referro-mcp/   CLI entrypoint (flags, env, stdio server)
internal/referro/  scraper client + models
internal/server/   MCP tool definitions + handlers
fixtures/          local site used by tests to stand in for referro.design
```
