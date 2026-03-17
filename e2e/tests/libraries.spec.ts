import { test, expect } from "@playwright/test";
import { createTestUser, signUp } from "./helpers/auth";

test.describe("Library management", () => {
  test("create a library from the dashboard and browse its empty books views", async ({
    page,
  }) => {
    const testUser = createTestUser({ displayName: "Library Test User" });
    const libraryName = `E2E Library ${Date.now()}`;
    const firstPath = `/tmp/e2e-library-${Date.now()}`;

    await signUp(page, testUser);

    await page.getByRole("button", { name: "Add Your First Library" }).click();
    await expect(page.getByRole("heading", { name: "Create Library" })).toBeVisible();

    await page.locator("form").getByRole("button", { name: "Create Library" }).click();
    await expect(page.getByText("Name is required")).toBeVisible();

    await page.locator("#lib-name").fill(libraryName);
    await page.locator("form").getByRole("button", { name: "Create Library" }).click();
    await expect(page.getByText("At least one folder is required")).toBeVisible();

    await page.locator("input[placeholder='/path/to/books']").first().fill(firstPath);
    await page.locator("form").getByRole("button", { name: "Create Library" }).click();

    await expect(page).toHaveURL(/\/#libraries\/[^/]+$/);
    await expect(page.getByRole("heading", { name: libraryName })).toBeVisible();
    await expect(page.getByText("No books yet.")).toBeVisible();
    await expect(
      page.locator("aside").getByRole("button", { name: libraryName }),
    ).toBeVisible();
    await expect(
      page.locator("aside").getByRole("button", { name: "All Books" }),
    ).toBeVisible();

    await page.locator("aside").getByRole("button", { name: "All Books" }).click();
    await expect(page).toHaveURL(/\/#books$/);
    await expect(page.getByRole("heading", { name: "All Books" })).toBeVisible();
    await expect(page.getByText("No books yet.")).toBeVisible();
  });
});
