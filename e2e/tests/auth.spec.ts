import { test, expect } from "@playwright/test";
import {
  configureTimeouts,
  createTestUser,
  getAuthErrorBanner,
  openAuthPage,
  openLoginForm,
  openSignupForm,
  signIn,
  signOut,
  signUp,
} from "./helpers/auth";

test.describe("Authentication flow", () => {
  test("register, verify dashboard, sign out, login again", async ({ page }) => {
    configureTimeouts(page);
    const testUser = createTestUser();

    await signUp(page, testUser);
    await expect(page.getByText("Get started with Biblioteka")).toBeVisible();
    await expect(page.getByText(testUser.email)).toBeVisible();

    await signOut(page);
    await signIn(page, testUser.email, testUser.password);

    await expect(page).toHaveURL("/");
    await expect(page.getByText("Get started with Biblioteka")).toBeVisible();
    await expect(page.getByText(testUser.email)).toBeVisible();
  });

  test("show validation and invalid credential errors", async ({ page }) => {
    configureTimeouts(page);
    const validationUser = createTestUser({ displayName: "Validation User" });
    const missingUser = createTestUser({ displayName: "Missing User" });
    const wrongPassword = "wrongpass123";

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

    await openLoginForm(page);
    await signIn(page, missingUser.email, wrongPassword);
    await expect(getAuthErrorBanner(page)).toContainText(/invalid email or password/i);
    await expect(page.getByRole("tab", { name: "Login", exact: true })).toBeVisible();
  });
});

test.describe("ARIA tabs accessibility", () => {
  test("tab roles, aria-selected, and tabpanel relationship", async ({ page }) => {
    configureTimeouts(page);
    await openAuthPage(page);

    const loginTab = page.locator("#login-tab");
    const signupTab = page.locator("#signup-tab");

    // Both buttons have role=tab
    await expect(loginTab).toHaveRole("tab");
    await expect(signupTab).toHaveRole("tab");

    // Tablist has aria-label
    await expect(page.getByRole("tablist")).toHaveAttribute("aria-label", "Authentication method");

    // Login is active by default
    await expect(loginTab).toHaveAttribute("aria-selected", "true");
    await expect(signupTab).toHaveAttribute("aria-selected", "false");
    await expect(loginTab).toHaveAttribute("tabindex", "0");
    await expect(signupTab).toHaveAttribute("tabindex", "-1");

    // Tabpanel points to the active tab
    const panel = page.getByRole("tabpanel");
    await expect(panel).toHaveAttribute("aria-labelledby", "login-tab");
    await expect(panel).toHaveAttribute("id", "login-panel");
    await expect(panel).toHaveAttribute("tabindex", "0");

    // Switch to Sign Up
    await signupTab.click();
    await expect(signupTab).toHaveAttribute("aria-selected", "true");
    await expect(loginTab).toHaveAttribute("aria-selected", "false");
    await expect(signupTab).toHaveAttribute("tabindex", "0");
    await expect(loginTab).toHaveAttribute("tabindex", "-1");
    await expect(panel).toHaveAttribute("aria-labelledby", "signup-tab");
    await expect(panel).toHaveAttribute("id", "signup-panel");
  });

  test("arrow key navigation between tabs", async ({ page }) => {
    configureTimeouts(page);
    await openAuthPage(page);

    const loginTab = page.locator("#login-tab");
    const signupTab = page.locator("#signup-tab");

    // Focus the login tab and press ArrowRight to move to signup
    await loginTab.focus();
    await page.keyboard.press("ArrowRight");
    await expect(signupTab).toBeFocused();
    await expect(signupTab).toHaveAttribute("aria-selected", "true");

    // Press ArrowLeft to go back to login
    await page.keyboard.press("ArrowLeft");
    await expect(loginTab).toBeFocused();
    await expect(loginTab).toHaveAttribute("aria-selected", "true");

    // Home goes to first tab, End goes to last tab
    await page.keyboard.press("End");
    await expect(signupTab).toBeFocused();
    await page.keyboard.press("Home");
    await expect(loginTab).toBeFocused();
  });
});
