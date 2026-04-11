# LaightDB

AI context database written in Go. Stores, searches, and retrieves context with minimal token usage.

## Architecture

```
LaightDB/
  cmd/laightdb/main.go           # Entry point (thin: wire deps, start server)
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
    server/
      http.go                    # REST API (net/http, no framework)
      middleware.go              # Logging, recovery
    mcp/
      server.go                  # MCP server setup + transport selection
      tools.go                   # MCP tool definitions
      resources.go               # MCP resource definitions
  Dockerfile                     # Multi-stage production build
  Dockerfile.dev                 # Dev image with air hot reload
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
    ID          string
    Collection  string
    Content     string
    ContentType string            // "code", "conversation", "doc", "kv"
    Summary     string
    Chunks      []Chunk
    Metadata    map[string]string
    Embedding   []float32         // 1024-dim from gobed
    CreatedAt   time.Time
    UpdatedAt   time.Time
    TokenCount  int
}
```

Serialized via custom binary codec (`internal/storage/codec.go`), not JSON.

## API Endpoints

- `POST   /v1/contexts`              -- Store context (auto-chunks, embeds, summarizes)
- `GET    /v1/contexts/{id}`         -- Get by ID (?detail=summary|full)
- `POST   /v1/search`               -- Hybrid search (query, filters, top_k, detail)
- `DELETE /v1/contexts/{id}`         -- Delete
- `GET    /v1/collections`           -- List collections
- `POST   /v1/collections/{name}/compact` -- Trigger storage compaction
- `GET    /v1/health`                -- Health check

## MCP Tools

- `store_context` -- Store content with metadata and collection
- `search_context` -- Hybrid search with query, filters, top_k
- `get_context` -- Retrieve by ID with detail level
- `delete_context` -- Remove context
- `list_collections` -- List collections
- `get_stats` -- Database stats

## Build & Test

```bash
go build ./cmd/laightdb              # Build binary
go test ./...                        # Run all tests
go test -race ./...                  # Race detector
go vet ./...                         # Static analysis
go tool golangci-lint run            # Linter
go fix ./...                         # Modernize code (Go 1.26)

docker compose build                 # Build production image
docker compose up -d                 # Run production
docker compose --profile dev up laightdb-dev  # Dev with hot reload
```

## Do NOT

- Add external dependencies without approval (especially ORMs, web frameworks, assertion libraries)
- Use `any` or `interface{}` when a concrete type or named interface works
- Put business logic in `cmd/` or handler layers
- Use global mutable state; pass dependencies via constructors
- Use `init()` functions
- Ignore errors or use `_ = someFunc()`
- Use `panic` for control flow
- Write comments that restate the code
