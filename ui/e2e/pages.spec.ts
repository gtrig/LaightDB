import { test, expect } from "@playwright/test";
import { installApiMock } from "./api-mock";

test.describe("main pages (open mode)", () => {
  test.beforeEach(async ({ page }) => {
    await installApiMock(page, "open");
  });

  test("dashboard shows stats and healthy status", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Dashboard", exact: true })).toBeVisible();
    await expect(page.getByText("Healthy")).toBeVisible();
    await expect(page.getByText("Entries", { exact: true })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: "ID" })).toBeVisible();
  });

  test("store page renders form", async ({ page }) => {
    await page.goto("/store");
    await expect(page.getByRole("heading", { name: "Store Context" })).toBeVisible();
    await expect(page.getByPlaceholder("Paste content here...")).toBeVisible();
  });

  test("store form submit navigates to new context", async ({ page }) => {
    await page.goto("/store");
    await page.getByPlaceholder("Select or type a new collection...").fill("e2e-col");
    await page.getByPlaceholder("Paste content here...").fill("hello world");
    await page.getByRole("button", { name: "Store Context" }).click();
    await expect(page).toHaveURL(/\/contexts\/new-id$/);
    await expect(page.getByText("new-id")).toBeVisible();
  });

  test("collections list and collection browse", async ({ page }) => {
    await page.goto("/collections");
    await expect(page.getByRole("heading", { name: "Collections" })).toBeVisible();
    const demoLink = page.getByRole("link", { name: "demo" });
    await expect(demoLink).toBeVisible();
    await demoLink.click();
    await expect(page).toHaveURL(/\/collections\/demo$/);
    await expect(page.getByRole("heading", { name: "demo" })).toBeVisible();
  });

  test("search page runs a query", async ({ page }) => {
    await page.goto("/search");
    await expect(page.getByRole("heading", { name: "Search" })).toBeVisible();
    await page.getByPlaceholder("Search contexts...").fill("test query");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await expect(page.getByText("Searching...")).not.toBeVisible({ timeout: 5000 });
  });

  test("system page shows System heading", async ({ page }) => {
    await page.goto("/system");
    await expect(page.getByRole("heading", { name: "System" })).toBeVisible();
  });

  test("3D explorer: empty graph message and engine labels", async ({ page }) => {
    await page.goto("/explorer");
    await expect(page.getByRole("heading", { name: "3D Explorer" })).toBeVisible();
    await expect(page.getByText("No graph data yet. Store some context entries and link them.")).toBeVisible();
    await page.getByRole("button", { name: "Engine Layout" }).click();
    await expect(page.getByText("WAL", { exact: true })).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText("MemTable", { exact: true })).toBeVisible();
  });

  test("stress test page loads preset queries", async ({ page }) => {
    await page.goto("/stress");
    await expect(page.getByRole("heading", { name: "Stress test" })).toBeVisible();
    await expect(page.getByRole("button", { name: /Show 2 queries/ })).toBeVisible();
    await page.getByRole("button", { name: /Show 2 queries/ }).click();
    await expect(page.locator("ul li", { hasText: "alpha" })).toBeVisible();
    await expect(page.locator("ul li", { hasText: "beta" })).toBeVisible();
  });
});
