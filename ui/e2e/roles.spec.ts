import { test, expect } from "@playwright/test";
import { installApiMock } from "./api-mock";

test.describe("admin user", () => {
  test.beforeEach(async ({ page }) => {
    await installApiMock(page, "admin");
  });

  test("shows Users and API Tokens in sidebar", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("link", { name: "Users" })).toBeVisible();
    await expect(page.getByRole("link", { name: "API Tokens" })).toBeVisible();
  });

  test("tokens page loads", async ({ page }) => {
    await page.goto("/settings/tokens");
    await expect(page.getByRole("heading", { name: "API Tokens" })).toBeVisible();
  });
});

test.describe("readonly user", () => {
  test.beforeEach(async ({ page }) => {
    await installApiMock(page, "readonly");
  });

  test("shows API Tokens but not Users", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("link", { name: "API Tokens" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Users" })).not.toBeVisible();
  });
});
