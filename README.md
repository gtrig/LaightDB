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
| `LAIGHTDB_SUMMARIZER` | `noop` (default), `openai`, `anthropic`, or `ollama` — used when storing context |
| `LAIGHTDB_OPENAI_BASE_URL` | OpenAI-compatible API root including `/v1` (default: `https://api.openai.com/v1`) |
| `LAIGHTDB_OPENAI_MODEL` | Chat model id (default: `gpt-4o-mini`) |
| `LAIGHTDB_OPENAI_API_KEY` | Required for the real OpenAI API; optional for local OpenAI-compatible servers |

See [AGENTS.md](AGENTS.md) for the full list as the binary is implemented.

### LM Studio (local LLM for summaries)

[LM Studio](https://lmstudio.ai/) exposes an **OpenAI-compatible** local server. Start a model and enable the local server (default is often port **1234**). Then run LaightDB with the OpenAI summarizer pointed at your machine:

```bash
export LAIGHTDB_SUMMARIZER=openai
export LAIGHTDB_OPENAI_BASE_URL=http://127.0.0.1:1234/v1
export LAIGHTDB_OPENAI_MODEL=your-model-id   # exact id shown in LM Studio for the loaded model
# API key not required for local LM Studio; omit or set a dummy if your client insists
./laightdb
```

If the server uses another host or port, change `LAIGHTDB_OPENAI_BASE_URL` accordingly (path must end with `/v1`).

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

## Authentication

LaightDB starts in **open mode** (no authentication) until the first user is created. Once a user exists, all non-health endpoints require authentication.

### Bootstrap an admin user

Set the environment variable before first start:

```bash
export LAIGHTDB_BOOTSTRAP_USER=admin:your-secure-password
```

Or create one via the API (no auth needed when no users exist):

```bash
curl -X POST http://localhost:8080/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"changeme","role":"admin"}'
```

### Web UI login

Navigate to the UI and sign in with username/password. A session cookie is set automatically.

### API / MCP authentication

Create an API token through the web UI (Settings → API Tokens) or via the API, then use it:

```bash
curl -H 'Authorization: Bearer ldb_...' http://localhost:8080/v1/contexts
```

MCP over HTTP uses the same Bearer token. MCP over stdio remains unauthenticated (local process).

### Rate limiting

Configurable per-user/per-IP token bucket rate limiting (default: 100 rps, burst 200). Returns `429 Too Many Requests` with `Retry-After` header when exceeded.

| Variable | Default | Purpose |
|----------|---------|---------|
| `LAIGHTDB_BOOTSTRAP_USER` | _(empty)_ | Seed admin user on first start (`username:password`) |
| `LAIGHTDB_SESSION_TTL` | `24h` | Session cookie lifetime |
| `LAIGHTDB_RATE_LIMIT_RPS` | `100` | Requests per second per key |
| `LAIGHTDB_RATE_LIMIT_BURST` | `200` | Burst capacity |

---

## REST API (optional)

Same capabilities as MCP, for scripts and integrations:

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/v1/contexts` | Create context |
| `GET` | `/v1/contexts` | List entries (`?collection=`, `?limit=`) |
| `GET` | `/v1/contexts/{id}` | Get by ID (`?detail=`) |
| `POST` | `/v1/search` | Hybrid search |
| `DELETE` | `/v1/contexts/{id}` | Delete |
| `GET` | `/v1/collections` | List collections |
| `GET` | `/v1/stats` | Database stats (entries, collections, vector nodes) |
| `POST` | `/v1/collections/{name}/compact` | Request compaction |
| `GET` | `/v1/health` | Health check |
| `POST` | `/v1/auth/login` | Login (returns session cookie) |
| `POST` | `/v1/auth/logout` | Logout (clears session) |
| `GET` | `/v1/auth/me` | Current user info |
| `GET` | `/v1/auth/status` | Auth required? (public) |
| `POST` | `/v1/users` | Create user |
| `GET` | `/v1/users` | List users |
| `DELETE` | `/v1/users/{id}` | Delete user |
| `PUT` | `/v1/users/{id}/password` | Change password |
| `PUT` | `/v1/users/{id}/role` | Change role |
| `POST` | `/v1/tokens` | Create API token |
| `GET` | `/v1/tokens` | List tokens |
| `DELETE` | `/v1/tokens/{id}` | Revoke token |

Example (once the server is running):

```bash
curl -s http://localhost:8080/v1/health
```

---

## Docker

When [Dockerfile](Dockerfile) and [docker-compose.yml](docker-compose.yml) exist:

Copy [`.env.example`](.env.example) to `.env` and edit ports or LLM settings, then:

```bash
cp .env.example .env
docker compose up -d
```

Compose reads `.env` for variable substitution (ports, summarizer, API keys). `.env` is gitignored. `extra_hosts` maps `host.docker.internal` to the host so containers can reach LM Studio or Ollama on your machine when you set the matching URLs.

Persist data with a volume mounted at `LAIGHTDB_DATA_DIR` (e.g. `/data` in the container). For MCP over HTTP from a host client, publish the HTTP port and set `LAIGHTDB_MCP_TRANSPORT=http` as documented in compose.

---

## Troubleshooting

- **First run slow / download:** gobed fetches model weights once; ensure outbound network or pre-seed the cache path under your data directory (documented when implementation is fixed).
- **Empty search results:** confirm content was stored in the same `collection` you filter on; check `get_stats` and logs.
- **MCP not connecting:** verify `command` is absolute, `LAIGHTDB_MCP_TRANSPORT=stdio`, and the binary is executable.

---

## License

Specify in [LICENSE](LICENSE) when added.
