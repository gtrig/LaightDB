import type {
  ContextEntry,
  SearchResult,
  StatsResponse,
  DetailLevel,
  EntryListItem,
  UserInfo,
  APITokenInfo,
  AuthStatus,
  UserRole,
  StressReport,
  StorageDiagnostics,
  GraphOverview,
} from "./types";

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...init,
    credentials: "include",
  });
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(`${res.status}: ${text}`);
  }
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  if (text === "") return undefined as T;
  return JSON.parse(text) as T;
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

export async function listEntries(options?: { collection?: string; limit?: number }): Promise<EntryListItem[]> {
  const q = new URLSearchParams();
  if (options?.collection) q.set("collection", options.collection);
  if (options?.limit != null) q.set("limit", String(options.limit));
  const qs = q.toString();
  const data = await request<{ entries: EntryListItem[] }>(`/v1/contexts${qs ? `?${qs}` : ""}`);
  return data.entries ?? [];
}

export async function getContext(
  id: string,
  detail: DetailLevel = "summary"
): Promise<ContextEntry> {
  return request(`/v1/contexts/${encodeURIComponent(id)}?detail=${detail}`);
}

export async function deleteContext(id: string): Promise<void> {
  return request(`/v1/contexts/${encodeURIComponent(id)}`, { method: "DELETE" });
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

export async function deleteCollection(name: string): Promise<{ deleted: number }> {
  return request(`/v1/collections/${encodeURIComponent(name)}`, { method: "DELETE" });
}

export async function compactCollection(name: string): Promise<void> {
  return request(`/v1/collections/${encodeURIComponent(name)}/compact`, { method: "POST" });
}

export async function getStressQueries(): Promise<string[]> {
  const data = await request<{ queries: string[] }>("/v1/stress/queries");
  return data.queries ?? [];
}

export async function runStress(body: {
  collection?: string;
  writes: number;
  write_concurrency: number;
  searches: number;
  search_concurrency: number;
  top_k: number;
  detail: string;
}): Promise<StressReport> {
  return request("/v1/stress", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

// --- Storage diagnostics ---

export async function getStorageDiagnostics(): Promise<StorageDiagnostics> {
  return request("/v1/storage/diagnostics");
}

// --- Graph overview ---

export async function getGraphOverview(options?: {
  collection?: string;
  limit?: number;
}): Promise<GraphOverview> {
  const q = new URLSearchParams();
  if (options?.collection) q.set("collection", options.collection);
  if (options?.limit != null) q.set("limit", String(options.limit));
  const qs = q.toString();
  return request(`/v1/graph/overview${qs ? `?${qs}` : ""}`);
}

// --- Auth ---

export async function getAuthStatus(): Promise<AuthStatus> {
  return request("/v1/auth/status");
}

export async function login(username: string, password: string): Promise<{ user: UserInfo }> {
  return request("/v1/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
}

export async function logout(): Promise<void> {
  return request("/v1/auth/logout", { method: "POST" });
}

export async function getMe(): Promise<{ user: UserInfo }> {
  return request("/v1/auth/me");
}

// --- Users ---

export async function listUsers(): Promise<UserInfo[]> {
  const data = await request<{ users: UserInfo[] }>("/v1/users");
  return data.users ?? [];
}

export async function createUser(username: string, password: string, role: UserRole): Promise<{ user: UserInfo }> {
  return request("/v1/users", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password, role }),
  });
}

export async function deleteUser(id: string): Promise<void> {
  return request(`/v1/users/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export async function changePassword(id: string, password: string): Promise<void> {
  return request(`/v1/users/${encodeURIComponent(id)}/password`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
}

export async function changeRole(id: string, role: UserRole): Promise<void> {
  return request(`/v1/users/${encodeURIComponent(id)}/role`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ role }),
  });
}

// --- Tokens ---

export async function listTokens(): Promise<APITokenInfo[]> {
  const data = await request<{ tokens: APITokenInfo[] }>("/v1/tokens");
  return data.tokens ?? [];
}

export async function createToken(name: string, role: UserRole): Promise<{ token: string; id: string; name: string; prefix: string; role: UserRole; created_at: string }> {
  return request("/v1/tokens", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, role }),
  });
}

export async function revokeToken(id: string): Promise<void> {
  return request(`/v1/tokens/${encodeURIComponent(id)}`, { method: "DELETE" });
}
