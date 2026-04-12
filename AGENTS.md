# LaightDB

AI context database written in Go. Stores, searches, and retrieves context with minimal token usage.

## Architecture

```
LaightDB/
  cmd/laightdb/main.go           # Entry point (thin: wire deps, start server)
  cmd/laightdb-dev-mcp/main.go   # Dev-only MCP stdio (debug data dir; not for production exposure)
  internal/
    config/config.go             # Env + flag configuration (no config file)
    storage/                     # Custom LSM-tree storage engine
      skiplist.go                # Skip list data structure
      bloom.go                   # Bloom filter
      codec.go                   # Binary encoder/decoder for ContextEntry
      engine.go                  # Orchestrator: WAL + MemTable + SSTables
      wal.go                     # Write-ahead log (append-only, crash recovery)
      memtable.go                # In-memory sorted map (skip list)
      sstable.go                 # Immutable on-disk sorted files + bloom filter
      compaction.go              # Background SSTable merging
    index/
      fulltext.go                # BM25 inverted index (tokenizer + scoring)
      vector.go                  # coder/hnsw wrapper (ANN search)
      metadata.go                # Inverted index on metadata key-value pairs
      hybrid.go                  # Reciprocal Rank Fusion combiner
      graph.go                   # In-memory bidirectional adjacency index (mindmap edges)
    context/
      tokens.go                  # Token count estimator (heuristic)
      store.go                   # Context CRUD + business logic
      chunker.go                 # Semantic content splitting (~512 tokens)
      dedup.go                   # SHA-256 content hash deduplication
      tiered.go                  # 3-level retrieval: metadata / summary / full
    embedding/engine.go          # gobed wrapper + LRU caching
    summarize/
      provider.go                # Summarizer interface
      openai.go                  # OpenAI provider
      anthropic.go               # Anthropic provider
      ollama.go                  # Ollama provider
      noop.go                    # No-op fallback (default)
    auth/
      auth.go                    # Role, User, APIToken, Session types, context helpers
      store.go                   # FileAuthStore: JSON persistence, user/token/session CRUD
      middleware.go              # Auth middleware (cookie + bearer, open mode, role enforcement)
      ratelimit.go               # Token bucket rate limiter (per-user/per-IP)
    server/
      http.go                    # REST API (net/http, no framework)
      graph_handlers.go          # Edge + graph REST endpoints
      auth_handlers.go           # Auth, user, token management endpoints
      middleware.go              # Logging, recovery
    mcp/
      server.go                  # MCP server setup + transport selection
      tools.go                   # MCP tool definitions
      resources.go               # MCP resource definitions
    mcpdev/
      server.go                  # Dev-only MCP: stdio or streamable HTTP (/mcp) via LAIGHTDB_DEV_MCP_HTTP_ADDR
      http.go                    # ListenAndServeHTTP for local debugging (not for production exposure)
      tools.go                   # debug_* tools; auth summary without hashes
      safe.go                    # Path confinement under data directory
  Dockerfile                     # Multi-stage production build
  Dockerfile.dev                 # Dev image with air hot reload
  Dockerfile.dev-mcp             # Dev MCP only (streamable HTTP :9090, same data volume as dev API)
  docker-compose.yml             # Prod + dev profiles
  .air.toml                      # Hot reload config
  .dockerignore
  go.mod
  go.sum
  Makefile
  README.md
```

## Dependencies

Only these external dependencies are approved:

| Package | Purpose |
|---|---|
| `github.com/modelcontextprotocol/go-sdk/mcp` | Official MCP SDK (stdio + streamable HTTP) |
| `github.com/lee101/gobed` | Built-in text embeddings (1024-dim static model, 119 MB weights) |
| `github.com/coder/hnsw` | HNSW vector index (in-memory, pure Go, persist/load) |
| `github.com/google/uuid` | UUID generation |
| `golang.org/x/crypto/bcrypt` | Password hashing (quasi-stdlib) |

Dev tools (tracked via `tool` directive in go.mod):

| Tool | Purpose |
|---|---|
| `github.com/golangci/golangci-lint/cmd/golangci-lint` | Linter (`go tool golangci-lint run`) |
| `github.com/air-verse/air` | Hot reload for dev Docker container |

Everything else (storage engine, BM25, chunking, binary codec) is built from scratch. Use stdlib wherever possible. Do NOT add dependencies without explicit approval.

## Conventions

- Go 1.26+, use `log/slog` for logging, `net/http` stdlib routing (Go 1.22+ mux patterns)
- Use `sync.WaitGroup.Go` for spawning tracked goroutines (Go 1.25+)
- Use `testing/synctest` for concurrent tests (Go 1.25+)
- Use `t.Context()` in tests for auto-canceled context (Go 1.24+)
- Use `new(expr)` for pointer-to-value init where useful (Go 1.26+)
- Track dev tools via `tool` directive in go.mod; run with `go tool <name>`
- All application code lives in `internal/` -- nothing in `pkg/`
- `cmd/laightdb/main.go` stays under 50 lines: wire deps, start server
- Interfaces define contracts between layers; implementations live in the same package
- Use `context.Context` as the first parameter of all public functions
- Errors: wrap with `fmt.Errorf("operation: %w", err)`, never discard
- Tests: stdlib `testing` package, files next to source (`foo_test.go`), use `t.Parallel()`
- No `testify`, no `gomock` -- use table-driven tests and hand-written fakes
- Naming: `storage.Engine`, not `storage.StorageEngine`; avoid stutter

## Data Model

```go
type ContextEntry struct {
    ID                string
    Collection        string
    Content           string
    ContentType       string            // "code", "conversation", "doc", "kv"
    Summary           string
    Chunks            []Chunk
    Metadata          map[string]string
    Embedding         []float32         // 1024-dim from gobed
    CreatedAt         time.Time
    UpdatedAt         time.Time
    TokenCount        int
    CompactContent    string
    CompactTokenCount int
}

// Edge connects two ContextEntry nodes (mindmap relationship).
type Edge struct {
    ID        string
    FromID    string
    ToID      string
    Label     string            // "child", "related_to", "depends_on", "auto_similar"
    Weight    float64           // user importance or cosine similarity (auto edges)
    Source    string            // "user" or "auto"
    Metadata  map[string]string
    CreatedAt time.Time
}
```

Both serialized via custom binary codecs (`internal/storage/codec.go`, `internal/storage/edge_codec.go`), not JSON.

### LSM Key Scheme

| Prefix | Purpose |
|---|---|
| `d:<id>` | ContextEntry document |
| `e:<edgeID>` | Canonical Edge record |
| `ef:<fromID>:<edgeID>` | Forward adjacency index (outgoing edges) |
| `et:<toID>:<edgeID>` | Reverse adjacency index (incoming edges) |

The `ef:` and `et:` prefix scans give O(degree) adjacency lookup without full-table scans.

## API Endpoints

- `POST   /v1/contexts`              -- Store context (auto-chunks, embeds, summarizes)
- `GET    /v1/contexts`               -- List entries (?collection=, ?limit=, newest first)
- `GET    /v1/contexts/{id}`         -- Get by ID (?detail=summary|full)
- `POST   /v1/search`               -- Hybrid search (query, filters, top_k, detail)
- `DELETE /v1/contexts/{id}`         -- Delete
- `GET    /v1/collections`           -- List collections
- `POST   /v1/collections/{name}/compact` -- Trigger storage compaction
- `GET    /v1/health`                -- Health check

### Graph / Edge Endpoints

- `POST   /v1/edges`                         -- Create edge (from_id, to_id, label, weight, source, metadata)
- `GET    /v1/edges?from=X`                  -- List outgoing edges from node X
- `GET    /v1/edges?to=X`                    -- List incoming edges to node X
- `GET    /v1/edges/{id}`                    -- Get edge by ID
- `DELETE /v1/edges/{id}`                    -- Delete edge
- `GET    /v1/graph/{id}/neighbors?depth=1`  -- BFS neighbors (both directions)
- `GET    /v1/graph/{id}/subtree?depth=3`    -- Directed BFS subtree (outgoing only)
- `POST   /v1/graph/search`                  -- 3-signal search (BM25 + vector + graph proximity)
- `GET    /v1/graph/{id}/suggest-links`      -- Vector-discovered link suggestions (?threshold=0.7&top_k=10)

### Auth & User Management

- `POST   /v1/auth/login`           -- Login with username/password, sets session cookie
- `POST   /v1/auth/logout`          -- Clear session cookie
- `GET    /v1/auth/me`              -- Current user info (from session or token)
- `GET    /v1/auth/status`          -- Whether auth is required (public)
- `POST   /v1/users`                -- Create user (first user bootstraps admin without auth)
- `GET    /v1/users`                -- List users (admin only)
- `DELETE /v1/users/{id}`           -- Delete user + cascade tokens/sessions (admin only)
- `PUT    /v1/users/{id}/password`  -- Change password (admin or self)
- `PUT    /v1/users/{id}/role`      -- Change role (admin only)
- `POST   /v1/tokens`               -- Create API token (returns plaintext once)
- `GET    /v1/tokens`               -- List tokens (own for readonly, all for admin)
- `DELETE /v1/tokens/{id}`          -- Revoke token

## Authentication

Two auth paths:
- **Web UI:** Username/password login → HTTP-only session cookie (`ldb_session`)
- **API/MCP:** `Authorization: Bearer <token>` header with API token

**Open mode:** When no users exist, all requests pass through unauthenticated (backward compatible). Creating the first user activates auth.

**Roles:** `admin` (full access) and `readonly` (GET + POST /v1/search only).

Auth data persisted as JSON in `{data_dir}/auth/` (users.json, tokens.json, sessions.json).

## MCP Tools

- `store_context` -- Store content with metadata and collection
- `search_context` -- Hybrid search with query, filters, top_k
- `get_context` -- Retrieve by ID with detail level
- `delete_context` -- Remove context
- `list_collections` -- List collections
- `get_stats` -- Database stats (includes edge count)

### Graph / Mindmap MCP Tools

- `link_context` -- Create a directed edge between two entries (from_id, to_id, label, weight, source)
- `unlink_context` -- Remove an edge by edge_id
- `get_neighbors` -- Get nodes connected to a given node via BFS (id, max_depth)
- `get_subtree` -- Return a mindmap subtree as structured JSON (id, max_depth)
- `graph_search` -- 3-signal search: BM25 + vector + graph proximity (query, focus_node_id, max_depth)
- `suggest_links` -- Auto-discover missing relationships via vector similarity (id, threshold, top_k)

## Build & Test

Development is **test-driven** (TDD): add or extend tests before production code; target high package coverage (see [.cursor/plans/laightdb_context_database_522a216f.plan.md](.cursor/plans/laightdb_context_database_522a216f.plan.md)).

```bash
go build ./cmd/laightdb              # Build binary
go test ./...                        # Run all tests
go test -cover ./...                 # Coverage summary per package
go test -race ./...                  # Race detector
go vet ./...                         # Static analysis
go tool golangci-lint run            # Linter
go fix ./...                         # Modernize code (Go 1.26)

docker compose build                 # Build production image
docker compose up -d                 # Run production
docker compose --profile dev up laightdb-dev  # Dev with hot reload
```

**Implementation plan:** [.cursor/plans/laightdb_context_database_522a216f.plan.md](.cursor/plans/laightdb_context_database_522a216f.plan.md)

## Do NOT

- Add external dependencies without approval (especially ORMs, web frameworks, assertion libraries)
- Use `any` or `interface{}` when a concrete type or named interface works
- Put business logic in `cmd/` or handler layers
- Use global mutable state; pass dependencies via constructors
- Use `init()` functions
- Ignore errors or use `_ = someFunc()`
- Use `panic` for control flow
- Write comments that restate the code
