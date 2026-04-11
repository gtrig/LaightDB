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

/**
 * Intercepts /v1/* so the UI can be exercised without a running LaightDB API.
 */
export async function installApiMock(page: Page, scenario: ApiMockScenario) {
  await page.route("**/v1/**", async (route) => {
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

    if (path === "/v1/health" && method === "GET") {
      return fulfillJson(route, { status: "ok" });
    }

    if (path === "/v1/stats" && method === "GET") {
      return fulfillJson(route, { entries: 0, collections: 0, vector_nodes: 0 });
    }

    if (path === "/v1/collections" && method === "GET") {
      return fulfillJson(route, { collections: ["demo"] });
    }

    if (path === "/v1/contexts" && method === "GET") {
      return fulfillJson(route, { entries: [] });
    }

    if (path === "/v1/search" && method === "POST") {
      return fulfillJson(route, { hits: [] });
    }

    if (path === "/v1/users" && method === "GET") {
      return fulfillJson(route, { users: [] });
    }

    if (path === "/v1/tokens" && method === "GET") {
      return fulfillJson(route, { tokens: [] });
    }

    const ctxMatch = /^\/v1\/contexts\/([^/]+)$/.exec(path);
    if (ctxMatch && method === "GET") {
      return fulfillJson(route, { ...sampleEntry, id: ctxMatch[1] });
    }

    if (path.startsWith("/v1/tokens/") && method === "DELETE") {
      return fulfillEmpty(route, 204);
    }

    if (path.startsWith("/v1/users/") && method === "DELETE") {
      return fulfillEmpty(route, 204);
    }

    if (path.match(/^\/v1\/users\/[^/]+\/password$/) && method === "PUT") {
      return fulfillEmpty(route, 204);
    }

    if (path.match(/^\/v1\/users\/[^/]+\/role$/) && method === "PUT") {
      return fulfillEmpty(route, 204);
    }

    if (path === "/v1/users" && method === "POST") {
      return fulfillJson(route, { user: { ...adminUser, id: "new-user" } });
    }

    if (path === "/v1/tokens" && method === "POST") {
      return fulfillJson(route, {
        token: "ldb_test_token",
        id: "tok-1",
        name: "test",
        prefix: "ldb_",
        role: "readonly",
        created_at: "2020-01-01T00:00:00Z",
      });
    }

    if (path.match(/^\/v1\/collections\/[^/]+\/compact$/) && method === "POST") {
      return fulfillEmpty(route, 202);
    }

    if (path === "/v1/contexts" && method === "POST") {
      return fulfillJson(route, { id: "new-id" });
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
    if (path === "/v1/health" && method === "GET") {
      return fulfillJson(route, { status: "ok" });
    }
    if (path === "/v1/stats" && method === "GET") {
      return fulfillJson(route, { entries: 0, collections: 0, vector_nodes: 0 });
    }
    if (path === "/v1/collections" && method === "GET") {
      return fulfillJson(route, { collections: [] });
    }
    if (path === "/v1/contexts" && method === "GET") {
      return fulfillJson(route, { entries: [] });
    }
    if (path === "/v1/search" && method === "POST") {
      return fulfillJson(route, { hits: [] });
    }
    if (path === "/v1/users" && method === "GET") {
      return fulfillJson(route, { users: [] });
    }
    if (path === "/v1/tokens" && method === "GET") {
      return fulfillJson(route, { tokens: [] });
    }
    const ctxMatch = /^\/v1\/contexts\/([^/]+)$/.exec(path);
    if (ctxMatch && method === "GET") {
      return fulfillJson(route, { ...sampleEntry, id: ctxMatch[1] });
    }
    if (process.env.DEBUG_E2E_MOCK) {
      console.warn("[e2e mock] unhandled", method, path);
    }
    return fulfillJson(route, {});
  });
}

export function installLoginSuccessMock(page: Page) {
  return installLoginScenarioMock(page, "success");
}
