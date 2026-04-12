---
name: 3D Storage UI Dual Views
overview: >-
  Bulk graph overview + storage diagnostics REST APIs, and a Vite/React 3D Explorer
  (context graph + engine layout). Completed; kept for traceability with the main plan.
todos:
  - id: engine-diagnostics-api
    content: Engine.Diagnostics + GET /v1/storage/diagnostics + tests
    status: completed
  - id: graph-overview-api
    content: GraphIndex.AllEdges + Store.GraphOverview + GET /v1/graph/overview + tests
    status: completed
  - id: ui-deps-r3f
    content: three / @react-three/fiber / drei / d3-force-3d in ui/package.json
    status: completed
  - id: ui-api-types
    content: ui/src/api.ts + types.ts for diagnostics and overview
    status: completed
  - id: ui-storage-explorer-page
    content: StorageExplorer3D at /explorer; Layout nav
    status: completed
  - id: docs-readme
    content: README UI + REST sections for explorer and new routes
    status: completed
isProject: false
---

# 3D storage visualization (implemented)

## REST

| Endpoint | Purpose |
|----------|---------|
| `GET /v1/storage/diagnostics` | JSON `EngineDiagnostics`: `data_dir`, `wal_bytes`, `mem_entries`, `sstables[]` |
| `GET /v1/graph/overview` | Bulk nodes + edges for the mindmap UI (`?collection=`, `?limit=`) |

Go: `storage.Engine.Diagnostics()`, `context.Store.StorageDiagnostics`, `Store.GraphOverview`, handlers in `internal/server/storage_handlers.go` and `internal/server/graph_handlers.go`.

## UI

- Route: **`/explorer`** (`ui/src/components/StorageExplorer3D.tsx`).
- Tabs: **Context Graph** (force layout + edges), **Engine Layout** (WAL / memtable / SST schematic).
- Client: `getStorageDiagnostics()`, `getGraphOverview()` in `ui/src/api.ts`.

## Note

npm dependencies for `ui/` are separate from the Go module allowlist in `AGENTS.md`; only add frontend packages under `ui/package.json` with explicit approval.
