import { test, expect } from "@playwright/test";
import {
  createTestUser,
  getAuthErrorBanner,
  signIn,
  signOut,
  signUp,
} from "./helpers/auth";

test.describe("Account settings", () => {
  test("change password and use the new credentials", async ({ page }) => {
    const testUser = createTestUser({ displayName: "Settings Test User" });
    const oldPassword = testUser.password;
    const newPassword = "updatedpassword456";
    const tooShortPassword = "short";

    await signUp(page, testUser);

    await page.locator("aside").getByRole("button", { name: "Settings" }).click();
    await expect(page).toHaveURL(/\/#settings$/);
    await expect(page.getByRole("heading", { name: "Settings" })).toBeVisible();
    await expect(page.locator("#email-display")).toHaveValue(testUser.email);

    await page.getByRole("button", { name: "Update Password" }).click();
    await expect(page.getByText("Current password is required")).toBeVisible();

    await page.locator("#current-password").fill(testUser.password);
    await page.locator("#new-password").fill(tooShortPassword);
    await page.locator("#confirm-password").fill(tooShortPassword);
    await page.getByRole("button", { name: "Update Password" }).click();
    await expect(page.getByText("New password must be at least 6 characters")).toBeVisible();

    await page.locator("#new-password").fill(newPassword);
    await page.locator("#confirm-password").fill(`${newPassword}-mismatch`);
    await page.getByRole("button", { name: "Update Password" }).click();
    await expect(page.getByText("Passwords do not match")).toBeVisible();

    await page.locator("#confirm-password").fill(newPassword);
    await page.getByRole("button", { name: "Update Password" }).click();
    await expect(page.getByText("Password updated successfully")).toBeVisible();
    await expect(page.locator("#current-password")).toHaveValue("");
    await expect(page.locator("#new-password")).toHaveValue("");
    await expect(page.locator("#confirm-password")).toHaveValue("");

    await signOut(page);

    await signIn(page, testUser.email, oldPassword);
    await expect(getAuthErrorBanner(page)).toContainText(/invalid email or password/i);

    await page.locator("input#email").fill(testUser.email);
    await page.locator("input#password").fill(newPassword);
    await page.getByRole("button", { name: "Sign In" }).click();
    await expect(page).toHaveURL("/");
    await expect(page.getByText(testUser.email)).toBeVisible();
  });
});
