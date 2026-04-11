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
  ID: string;
  Score: number;
}

export interface SearchHit extends SearchResult {
  entry?: ContextEntry;
}

export interface StatsResponse {
  entries: number;
  collections: number;
  vector_nodes: number;
}

export type DetailLevel = "metadata" | "summary" | "full";

export type ContentType = "code" | "conversation" | "doc" | "kv";
