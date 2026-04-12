import type { Page, Route } from "@playwright/test";

export type ApiMockScenario = "open" | "auth-required" | "admin" | "readonly";

const adminUser = {
  id: "u-admin",
  username: "admin",
  role: "admin" as const,
  created_at: "2020-01-01T00:00:00Z",
  updated_at: "2020-01-01T00:00:00Z",
};

const readOnlyUser = {
  id: "u-ro",
  username: "reader",
  role: "readonly" as const,
  created_at: "2020-01-01T00:00:00Z",
  updated_at: "2020-01-01T00:00:00Z",
};

const sampleEntry = {
  id: "ctx-1",
  collection: "demo",
  content_type: "doc",
  metadata: {},
  token_count: 10,
  created_at: "2020-01-15T12:00:00Z",
  updated_at: "2020-01-15T12:00:00Z",
};

function fulfillJson(route: Route, body: object, status = 200) {
  return route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body),
  });
}

function fulfillText(route: Route, status: number, body: string) {
  return route.fulfill({ status, contentType: "text/plain; charset=utf-8", body });
}

function fulfillEmpty(route: Route, status: number) {
  return route.fulfill({ status });
}

const stressPhase = {
  requested: 10,
  ok: 10,
  errors: 0,
  wall: 1e7,
  p50: 1e5,
  p95: 2e5,
  p99: 3e5,
  ops_per_sec: 1000,
};

/**
 * Routes that do not depend on auth scenario (same payloads for open, admin, login mock, etc.).
 * Returns true if fulfilled.
 */
async function tryFulfillSharedV1Route(route: Route): Promise<boolean> {
  const req = route.request();
  const url = new URL(req.url());
  const path = url.pathname;
  const method = req.method();

  if (path === "/v1/graph/overview" && method === "GET") {
    await fulfillJson(route, { nodes: [], edges: [], truncated: false });
    return true;
  }

  if (path === "/v1/storage/diagnostics" && method === "GET") {
    await fulfillJson(route, {
      data_dir: "/tmp/mock-data",
      wal_bytes: 4096,
      mem_entries: 3,
      sstables: [{ path: "00001.sst", bytes: 8192, seq: 1 }],
    });
    return true;
  }

  if (path === "/v1/stress/queries" && method === "GET") {
    await fulfillJson(route, { queries: ["alpha", "beta"] });
    return true;
  }

  if (path === "/v1/stress" && method === "POST") {
    await fulfillJson(route, {
      base_url: "http://127.0.0.1:8080",
      collection: "demo",
      writes: stressPhase,
      searches: { ...stressPhase, requested: 5, ok: 5, ops_per_sec: 500 },
      total_wall: 1.5e7,
    });
    return true;
  }

  if (path.match(/^\/v1\/collections\/[^/]+$/) && method === "DELETE") {
    await fulfillJson(route, { deleted: 0 });
    return true;
  }

  if (path.match(/^\/v1\/contexts\/[^/]+$/) && method === "DELETE") {
    await fulfillEmpty(route, 204);
    return true;
  }

  if (path === "/v1/health" && method === "GET") {
    await fulfillJson(route, { status: "ok" });
    return true;
  }

  if (path === "/v1/stats" && method === "GET") {
    await fulfillJson(route, { entries: 0, collections: 1, vector_nodes: 0, edges: 0 });
    return true;
  }

  if (path === "/v1/collections" && method === "GET") {
    await fulfillJson(route, { collections: ["demo"] });
    return true;
  }

  if (path === "/v1/contexts" && method === "GET") {
    await fulfillJson(route, { entries: [] });
    return true;
  }

  if (path === "/v1/search" && method === "POST") {
    await fulfillJson(route, { hits: [] });
    return true;
  }

  if (path === "/v1/users" && method === "GET") {
    await fulfillJson(route, { users: [] });
    return true;
  }

  if (path === "/v1/tokens" && method === "GET") {
    await fulfillJson(route, { tokens: [] });
    return true;
  }

  const ctxMatch = /^\/v1\/contexts\/([^/]+)$/.exec(path);
  if (ctxMatch && method === "GET") {
    await fulfillJson(route, { ...sampleEntry, id: ctxMatch[1] });
    return true;
  }

  if (path.startsWith("/v1/tokens/") && method === "DELETE") {
    await fulfillEmpty(route, 204);
    return true;
  }

  if (path.startsWith("/v1/users/") && method === "DELETE") {
    await fulfillEmpty(route, 204);
    return true;
  }

  if (path.match(/^\/v1\/users\/[^/]+\/password$/) && method === "PUT") {
    await fulfillEmpty(route, 204);
    return true;
  }

  if (path.match(/^\/v1\/users\/[^/]+\/role$/) && method === "PUT") {
    await fulfillEmpty(route, 204);
    return true;
  }

  if (path === "/v1/users" && method === "POST") {
    await fulfillJson(route, { user: { ...adminUser, id: "new-user" } });
    return true;
  }

  if (path === "/v1/tokens" && method === "POST") {
    await fulfillJson(route, {
      token: "ldb_test_token",
      id: "tok-1",
      name: "test",
      prefix: "ldb_",
      role: "readonly",
      created_at: "2020-01-01T00:00:00Z",
    });
    return true;
  }

  if (path.match(/^\/v1\/collections\/[^/]+\/compact$/) && method === "POST") {
    await fulfillEmpty(route, 202);
    return true;
  }

  if (path === "/v1/contexts" && method === "POST") {
    await fulfillJson(route, { id: "new-id" });
    return true;
  }

  return false;
}

/**
 * Intercepts /v1/* so the UI can be exercised without a running LaightDB API.
 */
export async function installApiMock(page: Page, scenario: ApiMockScenario) {
  await page.route("**/v1/**", async (route) => {
    if (await tryFulfillSharedV1Route(route)) return;

    const req = route.request();
    const url = new URL(req.url());
    const path = url.pathname;
    const method = req.method();

    if (path === "/v1/auth/status" && method === "GET") {
      if (scenario === "open") return fulfillJson(route, { auth_required: false });
      return fulfillJson(route, { auth_required: true });
    }

    if (path === "/v1/auth/me" && method === "GET") {
      if (scenario === "admin") return fulfillJson(route, { user: adminUser });
      if (scenario === "readonly") return fulfillJson(route, { user: readOnlyUser });
      return fulfillText(route, 401, "unauthorized");
    }

    if (path === "/v1/auth/logout" && method === "POST") {
      return fulfillEmpty(route, 204);
    }

    if (process.env.DEBUG_E2E_MOCK) {
      console.warn("[e2e mock] unhandled", method, path);
    }
    return fulfillJson(route, {});
  });
}

export type LoginApiOutcome = "success" | "failure";

/**
 * Auth required + 401 on /me; POST /v1/auth/login follows `loginOutcome` (cookie not modeled).
 */
export async function installLoginScenarioMock(page: Page, loginOutcome: LoginApiOutcome) {
  await page.route("**/v1/**", async (route) => {
    if (await tryFulfillSharedV1Route(route)) return;

    const req = route.request();
    const url = new URL(req.url());
    const path = url.pathname;
    const method = req.method();

    if (path === "/v1/auth/status" && method === "GET") {
      return fulfillJson(route, { auth_required: true });
    }
    if (path === "/v1/auth/me" && method === "GET") {
      return fulfillText(route, 401, "unauthorized");
    }
    if (path === "/v1/auth/login" && method === "POST") {
      if (loginOutcome === "failure") {
        return fulfillText(route, 401, "invalid credentials");
      }
      return fulfillJson(route, { user: adminUser });
    }
    if (path === "/v1/auth/logout" && method === "POST") {
      return fulfillEmpty(route, 204);
    }
    if (process.env.DEBUG_E2E_MOCK) {
      console.warn("[e2e mock] unhandled (login)", method, path);
    }
    return fulfillJson(route, {});
  });
}

export function installLoginSuccessMock(page: Page) {
  return installLoginScenarioMock(page, "success");
}
