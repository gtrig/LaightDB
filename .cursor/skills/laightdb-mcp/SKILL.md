---
name: laightdb-mcp
description: >-
  Stores and retrieves context in LaightDB via MCP tools (hybrid search, get by
  id, collections, stats). Use when the user connects the LaightDB MCP server in
  Cursor, wants to save project/session context for later, or asks to search or
  load previously stored LaightDB context through MCP.
---

# LaightDB MCP (client usage)

Implementation lives in `internal/mcp/` (production). The **dev-only** server (`laightdb-dev`, `internal/mcpdev/`) exposes `debug_*` tools only—use the production LaightDB MCP for `store_context` / `search_context` / `get_context`.

## Before calling tools

1. Confirm which MCP server name Cursor uses for this project (often `user-laightdb`). If unsure, list tools/resources for that server.
2. **Read the tool descriptor JSON** under the MCP folder (e.g. `mcps/<server>/tools/<tool>.json`) before the first call, so arguments match the schema exactly.

## Authentication

When the LaightDB instance has users, the HTTP API and `/mcp` use the same auth as REST: **Bearer API token** (or session cookie for browser). Configure the MCP connection in Cursor so requests include a valid token; in **open mode** (no users yet), unauthenticated access is allowed.

## Tools (production)

| Tool | Purpose |
|------|---------|
| `store_context` | Persist text; returns JSON `{"id":"<uuid>"}`. |
| `search_context` | Hybrid BM25 + vector search; returns `hits` plus token stats. |
| `get_context` | Fetch one entry by `id`. |
| `delete_context` | Remove by `id`; returns `{"ok":true}`. |
| `list_collections` | Lists collection names. |
| `get_stats` | Database statistics. |

### `store_context`

- **Required:** `collection` (namespace), `content` (raw text).
- **Optional:** `content_type` (`code`, `conversation`, `doc`, `kv`), `metadata` (string map).

Store stable labels in `metadata` (e.g. `project`, `topic`, `ticket`) so `search_context` can filter with `filters`.

### `search_context`

- **Required:** `query`.
- **Optional:** `collection`, `filters` (metadata equality), `top_k`, `detail`.

### `detail` (search and get)

Use one of: `metadata`, `summary`, `full`.

- **`metadata`:** ids, metadata, token counts, timestamps—smallest payload.
- **`summary`:** includes generated summary (default for `get_context` when `detail` is omitted).
- **`full`:** full stored content/chunks—use when the user needs verbatim text.

### `get_context`

- **Required:** `id`.
- **Optional:** `detail` (see above).

## Resource

- **`laightdb://collections`** — JSON list of collections (same data as `list_collections`).

## Suggested workflows

**Save for later**

1. `store_context` with a clear `collection` and useful `metadata`.
2. Keep the returned `id` if the user needs a direct link to that entry.

**Find again**

1. `search_context` with `query` (+ `collection` / `filters` as needed).
2. Start with `detail: "metadata"` or `"summary"` to save tokens; use `"full"` only when necessary.
3. For one entry, `get_context` with the `id` from a hit.

**Housekeeping**

- `list_collections` or read `laightdb://collections` to see namespaces.
- `delete_context` when the user explicitly wants removal.

## Content hygiene

Do not store secrets or credentials in `content` or `metadata`. Prefer references (paths, ticket IDs) over pasted tokens.
