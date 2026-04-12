import { test, expect } from "@playwright/test";
import { installApiMock } from "./api-mock";

test.describe("auth required", () => {
  test.beforeEach(async ({ page }) => {
    await installApiMock(page, "auth-required");
  });

  test("redirects unauthenticated users from dashboard to login", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole("textbox", { name: "Username" })).toBeVisible();
    await expect(page.getByLabel("Password")).toBeVisible();
  });

  test("login page shows LaightDB branding", async ({ page }) => {
    await page.goto("/login");
    await expect(page.getByText("LaightDB").first()).toBeVisible();
    await expect(page.getByRole("button", { name: "Sign In" })).toBeVisible();
  });

  test("redirects unauthenticated users from explorer to login", async ({ page }) => {
    await page.goto("/explorer");
    await expect(page).toHaveURL(/\/login$/);
  });
});
