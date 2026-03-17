import { test, expect } from "@playwright/test";
import {
  createTestUser,
  getAuthErrorBanner,
  openAuthPage,
  openSignupForm,
  signIn,
  signOut,
  signUp,
} from "./helpers/auth";

test.describe("Authentication flow", () => {
  test("register, verify dashboard, sign out, login again", async ({ page }) => {
    const testUser = createTestUser();

    await signUp(page, testUser);
    await signOut(page);
    await signIn(page, testUser.email, testUser.password);

    await expect(page).toHaveURL("/");
    await expect(page.getByText("Get started with Biblioteka")).toBeVisible();
    await expect(page.getByText(testUser.email)).toBeVisible();
  });

  test("show validation and invalid credential errors", async ({ page }) => {
    const validationUser = createTestUser({ displayName: "Validation User" });

    await openSignupForm(page);

    await page.getByRole("button", { name: "Create Account" }).click();
    await expect(getAuthErrorBanner(page)).toContainText("Please fill in all fields");

    await page.locator("input#name").fill(validationUser.displayName);
    await page.locator("input#email").fill(validationUser.email);
    await page.locator("input#password").fill("short");
    await page.getByRole("button", { name: "Create Account" }).click();
    await expect(getAuthErrorBanner(page)).toContainText(
      "Password must be at least 6 characters",
    );

    await openAuthPage(page);
    await signIn(page, `missing-${Date.now()}@example.com`, "wrongpass123");
    await expect(getAuthErrorBanner(page)).toContainText(/invalid email or password/i);
    await expect(page.getByRole("button", { name: "Login", exact: true })).toBeVisible();
  });
});
