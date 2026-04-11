# LaightDB

AI context database written in Go. Stores, searches, and retrieves context with minimal token usage.

## Architecture

```
LaightDB/
  cmd/laightdb/main.go           # Entry point (thin: wire deps, start server)
  internal/
    config/config.go             # YAML configuration
    storage/                     # Custom LSM-tree storage engine
      engine.go                  # Orchestrator: WAL + MemTable + SSTables
      wal.go                     # Write-ahead log (append-only, crash recovery)
      memtable.go                # In-memory sorted map (skip list)
      sstable.go                 # Immutable on-disk sorted files + bloom filter
      compaction.go              # Background SSTable merging
    index/
      fulltext.go                # BM25 inverted index (tokenizer + scoring)
      hnsw.go                    # HNSW vector index (multi-layer graph ANN)
      metadata.go                # Inverted index on metadata key-value pairs
      hybrid.go                  # Reciprocal Rank Fusion combiner
    context/
      store.go                   # Context CRUD + business logic
      chunker.go                 # Semantic content splitting (~512 tokens)
      dedup.go                   # SHA-256 content hash deduplication
      tiered.go                  # 3-level retrieval: metadata / summary / full
    embedding/engine.go          # gobed wrapper + caching
    summarize/
      provider.go                # Summarizer interface
      openai.go                  # OpenAI provider
      anthropic.go               # Anthropic provider
      ollama.go                  # Ollama provider
    server/
      http.go                    # REST API (net/http, no framework)
      middleware.go              # Logging, recovery, auth
    mcp/
      server.go                  # MCP server setup + transport selection
      tools.go                   # MCP tool definitions
      resources.go               # MCP resource definitions
  go.mod
  go.sum
```

## Dependencies

Only these external dependencies are approved:

| Package | Purpose |
|---|---|
| `github.com/modelcontextprotocol/go-sdk/mcp` | Official MCP SDK (stdio + streamable HTTP) |
| `github.com/lee101/gobed` | Built-in text embeddings (1024-dim static model) |
| `github.com/google/uuid` | UUID generation |
| `log/slog` | Structured logging (stdlib) |

Everything else (storage engine, BM25, HNSW, chunking) is built from scratch. Use stdlib wherever possible. Do NOT add dependencies without explicit approval.

## Conventions

- Go 1.22+, use `log/slog` for logging, `net/http` stdlib routing (Go 1.22 mux)
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

## API Endpoints

- `POST   /v1/contexts`              -- Store context (auto-chunks, embeds, summarizes)
- `GET    /v1/contexts/:id`          -- Get by ID (?detail=summary|full)
- `POST   /v1/search`                -- Hybrid search (query, filters, top_k, detail)
- `DELETE /v1/contexts/:id`          -- Delete
- `GET    /v1/collections`           -- List collections
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
go build ./cmd/laightdb          # Build binary
go test ./...                    # Run all tests
go test -race ./...              # Race detector
go vet ./...                     # Static analysis
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
