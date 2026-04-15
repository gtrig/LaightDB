export interface ContextEntry {
  id: string;
  collection: string;
  content_type: string;
  metadata: Record<string, string>;
  token_count: number;
  created_at: string;
  updated_at: string;
  content?: string;
  summary?: string;
  chunks?: Chunk[];
}

export interface Chunk {
  Index: number;
  ParentID: string;
  Text: string;
}

export interface SearchResult {
  id: string;
  score: number;
  token_count: number;
}

export interface SearchHit extends SearchResult {
  entry?: ContextEntry;
}

export interface StatsResponse {
  entries: number;
  collections: number;
  vector_nodes: number;
  edges?: number;
}

// --- Storage diagnostics ---

export interface SSTFileInfo {
  path: string;
  bytes: number;
  seq: number;
}

export interface StorageDiagnostics {
  data_dir: string;
  wal_bytes: number;
  mem_entries: number;
  sstables: SSTFileInfo[];
}

// --- Graph overview ---

export interface OverviewNode {
  id: string;
  collection: string;
  label: string;
}

export interface OverviewEdge {
  edge_id: string;
  from_id: string;
  to_id: string;
  label: string;
  weight: number;
  source: string;
}

export interface GraphOverview {
  nodes: OverviewNode[];
  edges: OverviewEdge[];
  truncated: boolean;
}

/** Row from GET /v1/contexts */
export interface EntryListItem {
  id: string;
  collection: string;
  content_type: string;
  token_count: number;
  created_at: string;
  updated_at: string;
}

export type DetailLevel = "metadata" | "summary" | "full";

export type ContentType = "code" | "conversation" | "doc" | "kv";

export type UserRole = "admin" | "readonly";

export interface UserInfo {
  id: string;
  username: string;
  role: UserRole;
  created_at: string;
  updated_at: string;
}

export interface APITokenInfo {
  id: string;
  user_id: string;
  name: string;
  prefix: string;
  role: UserRole;
  created_at: string;
  active: boolean;
  revoked_at?: string;
}

export interface AuthStatus {
  auth_required: boolean;
}

/** JSON durations are nanoseconds (Go time.Duration). */
export interface StressPhaseStat {
  requested: number;
  ok: number;
  errors: number;
  wall: number;
  p50: number;
  p95: number;
  p99: number;
  ops_per_sec: number;
}

export interface StressReport {
  base_url: string;
  collection: string;
  writes: StressPhaseStat;
  searches: StressPhaseStat;
  total_wall: number;
}

/** Row from GET /v1/audit/calls */
export interface CallLogEntry {
  id: string;
  ts: string;
  duration_ms: number;
  channel: "mcp";
  user_id?: string;
  username?: string;
  method?: string;
  path?: string;
  query?: string;
  tool?: string;
  status?: number;
  ok?: boolean;
  request?: string;
  response?: string;
}
