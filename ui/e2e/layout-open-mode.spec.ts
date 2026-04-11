import { test, expect } from "@playwright/test";
import { installApiMock } from "./api-mock";

test.describe("open mode (no users)", () => {
  test.beforeEach(async ({ page }) => {
    await installApiMock(page, "open");
  });

  test("sidebar shows Settings and Users for bootstrap", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("link", { name: "Dashboard" })).toBeVisible();
    await expect(page.getByText("Settings", { exact: true })).toBeVisible();
    await expect(page.getByRole("link", { name: "Users" })).toBeVisible();
    await expect(page.getByRole("link", { name: "API Tokens" })).not.toBeVisible();
  });

  test("Users page loads without redirect", async ({ page }) => {
    await page.goto("/settings/users");
    await expect(page.getByRole("heading", { name: "Users" })).toBeVisible();
  });
});
