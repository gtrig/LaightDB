import { test, expect } from "@playwright/test";
import { installApiMock } from "./api-mock";

test.describe("navigation", () => {
  test.beforeEach(async ({ page }) => {
    await installApiMock(page, "open");
  });

  test("Search route shows Search heading", async ({ page }) => {
    await page.goto("/search");
    await expect(page.getByRole("heading", { name: "Search" })).toBeVisible();
  });

  test("System route shows System heading", async ({ page }) => {
    await page.goto("/system");
    await expect(page.getByRole("heading", { name: "System" })).toBeVisible();
  });
});
