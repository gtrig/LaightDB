---
name: ui-web-explorer
description: >-
  Vite/React UI for LaightDB: dashboard, search, and the 3D Explorer (/explorer)
  for context graphs and storage engine layout. Use when editing ui/src, Vite
  config, or REST client calls for graph overview / storage diagnostics.
---

# LaightDB Web UI

## Layout

- **Entry:** `ui/src/main.tsx`, routes in `ui/src/App.tsx`.
- **API helpers:** `ui/src/api.ts` — includes `getGraphOverview()`, `getStorageDiagnostics()`.
- **Types:** `ui/src/types.ts` — `GraphOverview`, `StorageDiagnostics`, etc.
- **3D page:** `ui/src/components/StorageExplorer3D.tsx` — tabs for **Context Graph** (force layout) and **Engine Layout** (WAL / memtable / SST blocks).

## Backend endpoints used

| Purpose | Method | Path |
|---------|--------|------|
| Bulk graph for 3D mindmap | `GET` | `/v1/graph/overview?collection=&limit=` |
| WAL / SST sizes | `GET` | `/v1/storage/diagnostics` |

## Dev workflow

1. Start LaightDB with HTTP (e.g. `:8080`).
2. `cd ui && npm run dev` — requests to `/v1` proxy to the API.

Do not add Go dependencies for UI work; npm deps belong in `ui/package.json` with project approval.
