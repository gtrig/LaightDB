# LaightDB

Standalone Go database for **AI context**: store text, search with hybrid **full-text + semantic** retrieval, and return **tiered detail** (metadata / summary / full) to save context window. Exposes the same operations over **REST** and **MCP** (Model Context Protocol).

**Docs:** [AGENTS.md](AGENTS.md) (conventions) · [Implementation plan](.cursor/plans/laightdb_context_database_522a216f.plan.md) (TDD, coverage, phases)

---

## MCP-first usage

LaightDB is designed so assistants (Cursor, Claude Desktop, any MCP client) can **store** and **retrieve** context **without** calling HTTP directly. The REST API exists for automation, debugging, and non-MCP clients.

### Implementation gate (before calling the project “usable”)

Do not treat core development as complete until **both** are true:

1. **stdio MCP:** With `LAIGHTDB_MCP_TRANSPORT=stdio`, a client can run the binary as a subprocess and successfully call:
   - `store_context` — persist new content and receive an ID
   - `search_context` — run a hybrid query and receive ranked hits
   - `get_context` — load a stored entry by ID (at least `detail=summary`)
2. **Automated check:** Add at least one test or script that exercises the above path (even if it uses a test harness instead of a full MCP client).

Streamable HTTP MCP can follow; **stdio is the minimum bar** for local assistant integration.

---

## Requirements

- **Go 1.26+**
- **Disk:** data directory for the LSM engine and indexes (default `./data`)
- **Embeddings:** first run downloads the [gobed](https://github.com/lee101/gobed) static model (~119 MB) unless cached

---

## Build

```bash
go build -o laightdb ./cmd/laightdb
```

(After the implementation lands — see the [implementation plan](.cursor/plans/laightdb_context_database_522a216f.plan.md).)

---

## Configuration

Configuration is **environment variables and flags only** (no config file). Common variables:

| Variable | Purpose |
|----------|---------|
| `LAIGHTDB_DATA_DIR` | Database files (default `./data`) |
| `LAIGHTDB_HTTP_ADDR` | REST listen address (e.g. `:8080`) |
| `LAIGHTDB_MCP_TRANSPORT` | `stdio` (default) or `http` (streamable MCP over HTTP) |

See [AGENTS.md](AGENTS.md) for the full list as the binary is implemented.

---

## Running with MCP (stdio)

Recommended for Cursor, Claude Desktop, and other tools that spawn an MCP server process.

```bash
export LAIGHTDB_DATA_DIR=./data
export LAIGHTDB_MCP_TRANSPORT=stdio
./laightdb
```

The process speaks MCP over **stdin/stdout** (newline-delimited JSON-RPC). Your IDE or client starts this binary and connects automatically when configured.

### MCP tools (store / search / retrieve)

| Tool | Purpose |
|------|---------|
| `store_context` | Store content with `collection`, `content`, `content_type`, optional `metadata` |
| `search_context` | Hybrid search: `query`, optional `collection`, `filters`, `top_k`, `detail` |
| `get_context` | Get one entry by `id` and `detail` (`metadata`, `summary`, or `full`) |
| `delete_context` | Delete by `id` |
| `list_collections` | List collection names |
| `get_stats` | Database statistics |

**Typical assistant workflow**

1. **Store:** `store_context` after important facts or code context.
2. **Find:** `search_context` with a natural-language `query`.
3. **Open:** `get_context` with the returned `id` and `detail` as needed (prefer `summary` in the model context; use `full` only when necessary).

---

## Cursor MCP configuration

Add a server entry (paths adjusted to your clone):

```json
{
  "mcpServers": {
    "laightdb": {
      "command": "/absolute/path/to/laightdb",
      "env": {
        "LAIGHTDB_DATA_DIR": "/absolute/path/to/laightdb-data",
        "LAIGHTDB_MCP_TRANSPORT": "stdio"
      }
    }
  }
}
```

Restart Cursor after editing MCP settings. The exact file location depends on your Cursor version; use **Cursor Settings → MCP** or the documented `mcp.json` path for your OS.

---

## Claude Desktop (example)

```json
{
  "mcpServers": {
    "laightdb": {
      "command": "/absolute/path/to/laightdb",
      "env": {
        "LAIGHTDB_DATA_DIR": "/absolute/path/to/laightdb-data",
        "LAIGHTDB_MCP_TRANSPORT": "stdio"
      }
    }
  }
}
```

---

## MCP over HTTP (streamable transport)

For remote or containerized setups, run with `LAIGHTDB_MCP_TRANSPORT=http` and expose the MCP handler URL (implementation will document the exact path, typically mounted alongside REST). Clients that support streamable HTTP MCP can point at that URL instead of spawning stdio.

---

## REST API (optional)

Same capabilities as MCP, for scripts and integrations:

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/v1/contexts` | Create context |
| `GET` | `/v1/contexts/{id}` | Get by ID (`?detail=`) |
| `POST` | `/v1/search` | Hybrid search |
| `DELETE` | `/v1/contexts/{id}` | Delete |
| `GET` | `/v1/collections` | List collections |
| `POST` | `/v1/collections/{name}/compact` | Request compaction |
| `GET` | `/v1/health` | Health check |

Example (once the server is running):

```bash
curl -s http://localhost:8080/v1/health
```

---

## Docker

When [Dockerfile](Dockerfile) and [docker-compose.yml](docker-compose.yml) exist:

```bash
docker compose up -d
```

Persist data with a volume mounted at `LAIGHTDB_DATA_DIR` (e.g. `/data` in the container). For MCP over HTTP from a host client, publish the HTTP port and set `LAIGHTDB_MCP_TRANSPORT=http` as documented in compose.

---

## Troubleshooting

- **First run slow / download:** gobed fetches model weights once; ensure outbound network or pre-seed the cache path under your data directory (documented when implementation is fixed).
- **Empty search results:** confirm content was stored in the same `collection` you filter on; check `get_stats` and logs.
- **MCP not connecting:** verify `command` is absolute, `LAIGHTDB_MCP_TRANSPORT=stdio`, and the binary is executable.

---

## License

Specify in [LICENSE](LICENSE) when added.
