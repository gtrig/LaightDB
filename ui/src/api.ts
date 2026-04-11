import type { ContextEntry, SearchResult, StatsResponse, DetailLevel } from "./types";

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(`${res.status}: ${text}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export async function healthCheck(): Promise<{ status: string }> {
  return request("/v1/health");
}

export async function getStats(): Promise<StatsResponse> {
  return request("/v1/stats");
}

export async function listCollections(): Promise<string[]> {
  const data = await request<{ collections: string[] }>("/v1/collections");
  return data.collections ?? [];
}

export async function storeContext(body: {
  collection: string;
  content: string;
  content_type: string;
  metadata?: Record<string, string>;
}): Promise<{ id: string }> {
  return request("/v1/contexts", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export async function getContext(
  id: string,
  detail: DetailLevel = "summary"
): Promise<ContextEntry> {
  return request(`/v1/contexts/${id}?detail=${detail}`);
}

export async function deleteContext(id: string): Promise<void> {
  return request(`/v1/contexts/${id}`, { method: "DELETE" });
}

export async function searchContexts(body: {
  query: string;
  collection?: string;
  filters?: Record<string, string>;
  top_k?: number;
  detail?: string;
}): Promise<SearchResult[]> {
  const data = await request<{ hits: SearchResult[] }>("/v1/search", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return data.hits ?? [];
}

export async function compactCollection(name: string): Promise<void> {
  return request(`/v1/collections/${name}/compact`, { method: "POST" });
}
