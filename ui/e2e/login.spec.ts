import { test, expect } from "@playwright/test";
import { installLoginScenarioMock } from "./api-mock";

test.describe("login page", () => {
  test("successful submit navigates to dashboard and shows session in sidebar", async ({ page }) => {
    await installLoginScenarioMock(page, "success");
    await page.goto("/login");

    await page.getByRole("textbox", { name: "Username" }).fill("admin");
    await page.getByLabel("Password").fill("secret");
    await page.getByRole("button", { name: "Sign In" }).click();

    await expect(page).toHaveURL("/");
    await expect(page.getByRole("heading", { name: "Dashboard", exact: true })).toBeVisible();
    await expect(page.locator("aside").getByText("admin", { exact: true }).first()).toBeVisible();
    await expect(page.getByRole("link", { name: "API Tokens" })).toBeVisible();
  });

  test("failed login keeps user on login route and shows an error alert", async ({ page }) => {
    await installLoginScenarioMock(page, "failure");
    await page.goto("/login");

    await page.getByRole("textbox", { name: "Username" }).fill("admin");
    await page.getByLabel("Password").fill("wrong");
    await page.getByRole("button", { name: "Sign In" }).click();

    await expect(page).toHaveURL("/login");
    const alert = page.getByRole("alert");
    await expect(alert).toBeVisible();
    await expect(alert).toContainText("401");
    await expect(alert).toContainText("invalid credentials");
  });

  test("shows Signing in while the request is in flight", async ({ page }) => {
    await installLoginScenarioMock(page, "failure");
    await page.route("**/v1/auth/login", async (route) => {
      await new Promise((r) => setTimeout(r, 300));
      await route.fulfill({
        status: 401,
        contentType: "text/plain; charset=utf-8",
        body: "slow",
      });
    });

    await page.goto("/login");
    await page.getByRole("textbox", { name: "Username" }).fill("x");
    await page.getByLabel("Password").fill("y");

    const click = page.getByRole("button", { name: "Sign In" }).click();
    await expect(page.getByRole("button", { name: "Signing in…" })).toBeVisible();
    await click;

    await expect(page.getByRole("button", { name: "Sign In" })).toBeVisible();
  });

  test("HTML required fields block submit when empty", async ({ page }) => {
    await installLoginScenarioMock(page, "success");
    await page.goto("/login");

    await page.getByRole("button", { name: "Sign In" }).click();

    await expect(page).toHaveURL("/login");
    await expect(page.locator("[role=alert]")).toHaveCount(0);
  });
});
