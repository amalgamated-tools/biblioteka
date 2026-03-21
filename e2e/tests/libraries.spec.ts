import { test, expect } from "@playwright/test";
import { configureTimeouts } from "./helpers/auth";
import { signInAsAdmin } from "./helpers/admin";

test.describe("Library management", () => {
  test("create a library and verify it appears in the UI", async ({ page }) => {
    configureTimeouts(page);

    await signInAsAdmin(page);

    // Open the library creation form via the sidebar's "Create library" link.
    await page
      .locator("aside")
      .getByRole("link", { name: "Create library" })
      .click();
    await page.waitForSelector("#lib-name");

    const libraryName = `E2E Library ${Date.now()}`;

    // Fill in the required fields.
    await page.locator("#lib-name").fill(libraryName);
    await page
      .locator('input[aria-label="Folder path"]')
      .fill("/tmp");

    // Submit the form.
    await page.getByRole("button", { name: "Create Library" }).click();

    // After a successful save the app navigates to the library view.
    await expect(page).toHaveURL(/#libraries\/.+/);

    // The library name should be displayed as the page heading.
    await expect(page.locator("h1")).toContainText(libraryName);

    // The library should also appear in the sidebar navigation list.
    await expect(
      page.locator("aside").getByText(libraryName),
    ).toBeVisible();
  });

  test("show validation errors when the library form is incomplete", async ({
    page,
  }) => {
    configureTimeouts(page);

    await signInAsAdmin(page);

    // Open the creation form.
    await page
      .locator("aside")
      .getByRole("link", { name: "Create library" })
      .click();
    await page.waitForSelector("#lib-name");

    // Submit without filling in any field.
    await page.getByRole("button", { name: "Create Library" }).click();
    await expect(page.locator("#lib-name-error")).toContainText(
      "Name is required",
    );

    // Fill the name but leave the folder path empty.
    await page.locator("#lib-name").fill("Incomplete Library");
    await page.getByRole("button", { name: "Create Library" }).click();
    await expect(page.locator("#lib-folders-error")).toContainText(
      "At least one folder is required",
    );
  });
});
