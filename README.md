# referro-mcp

A local [MCP](https://modelcontextprotocol.io) server that gives coding agents
access to [refero.design](https://refero.design)'s catalog of real product
designs — captured screens of websites and iOS apps, plus the multi-step user
flows behind them.

It runs as a local CLI speaking MCP over stdio, so any MCP client can use it —
including coding agents like **OpenCode** and **KiloCode**.

## Why this exists

Refero ships an official MCP server, but it requires a **paid plan** (Pro,
Team, or Lifetime). Refero also exposes a **free public JSON API**
(`api.refero.design/v1`) that covers the same core data — searchable screens,
flows, and design patterns — with no auth. This server wraps that free public
API, giving agents the same design-research capability at no cost.

## Tools

The server registers four tools:

| Tool | Description |
|------|-------------|
| `list_designs` | Search the catalog of captured designs (websites or iOS apps) by query, industry, or visual direction. Returns screens with site/app, page URL, colors, fonts, page types, and image URLs. |
| `get_design` | Fetch a single design (screen) by UUID, site/app name or domain, or numeric screen ID. Returns full image URLs, page URL, colors, fonts, page types, patterns, and associated flow IDs. |
| `get_design_images` | Fetch the image URLs for a design (full-resolution, thumbnail, preview). |
| `get_design_workflow` | Fetch a workflow (a flow: a multi-step user journey like checkout, onboarding, or password reset) by its numeric ID, with the ordered step screens. |

`get_design`'s `ref` accepts a screen UUID, a site/app name or domain (e.g.
`stripe.com`), or a numeric screen ID, so agents can pass whatever they got
from `list_designs`.

## Build

Requires Go 1.25+.

```sh
go build -o referro-mcp ./cmd/referro-mcp
```

## Run

```sh
./referro-mcp
```

Config via flags or environment variables:

| Flag | Env | Default | Purpose |
|------|-----|---------|---------|
| `-base-url` | `REFERRO_BASE_URL` | `https://api.refero.design/v1` | API base URL |
| `-cache-dir` | `REFERRO_CACHE_DIR` | *(off)* | Disk cache directory |
| `-cache-ttl` | — | `10m` | Cache freshness window |

The disk cache is worth enabling for agent use: coding agents re-invoke tools
often, and the cache stops every call from hitting the API again.

```sh
./referro-mcp -cache-dir ~/.cache/referro-mcp
```

## Test

The test suite serves captured API responses from `fixtures/api/` (real
responses from the live API), so tests are deterministic and run offline.

```sh
go test ./...
```

## Wire it into a coding agent

### OpenCode

```sh
opencode mcp add referro -- ./referro-mcp
```

This writes an entry to `~/.config/opencode/opencode.jsonc`:

```jsonc
{
  "mcp": {
    "referro": {
      "type": "local",
      "command": ["/absolute/path/to/referro-mcp"]
    }
  }
}
```

Verify with `opencode mcp list` (should show `referro connected`).

### KiloCode

KiloCode is an OpenCode fork with the same CLI, so the config is identical
under `~/.config/kilo/kilo.jsonc`:

```sh
kilocode mcp add referro -- ./referro-mcp
```

### Any other MCP client

Point it at the binary as a stdio command. For example, Claude Desktop:

```json
{
  "mcpServers": {
    "referro": {
      "command": "/absolute/path/to/referro-mcp"
    }
  }
}
```

## API notes

The free public API is undocumented, so this client models the endpoints it
actually needs. Observed endpoints (all `GET`, no auth):

- `/v1/search?q=&page=&per_page=` — website screens
- `/v1/search/apps?q=&page=&per_page=` — iOS app screens
- `/v1/screenshots/{uuid}` — single screen
- `/v1/screenshots/{id}/similar?page=` — visually similar screens
- `/v1/flows/{id}` — a flow's ordered steps
- `/v1/sites/{id}`, `/v1/apps/{id}` — product metadata
- `/v1/design_patterns/available` — filterable design patterns

The API is inconsistent about types across endpoints (e.g. `url` is a string
on some routes and an array on others; `fonts` is a string array on some and an
object array on others). The client's `StringList` and `FontList` types absorb
both shapes.

## Layout

```
cmd/referro-mcp/   CLI entrypoint (flags, env, stdio server)
internal/referro/  API client + models (tolerant JSON types)
internal/server/   MCP tool definitions + handlers
fixtures/api/      captured live-API responses used by tests
```
