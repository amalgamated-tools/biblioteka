import { test, expect, type Page } from "@playwright/test";

const AUTH_ERROR_TEST_ID = "auth-error";

const DEFAULT_TIMEOUT_MS = Number(process.env.SCREENSHOT_TIMEOUT_MS || 5000);
const NAVIGATION_TIMEOUT_MS = Number(process.env.SCREENSHOT_NAVIGATION_TIMEOUT_MS || 5000);

async function openAuthPage(page: Page): Promise<void> {
  await page.goto(`/`, {
    waitUntil: 'networkidle',
    timeout: NAVIGATION_TIMEOUT_MS,
  });
  await page.waitForSelector('button#login-btn');
}

async function openSignupForm(page: Page): Promise<void> {
  await openAuthPage(page);
  await page.getByRole('button', { name: 'Sign Up', exact: true }).click();
  await page.waitForSelector('input#name');
  await page.waitForFunction(() => {
    const btn = document.querySelector('button[type="submit"]');
    return btn && btn.textContent.trim() === 'Create Account';
  });
}

export function getAuthErrorBanner(page: Page) {
  return page.getByTestId(AUTH_ERROR_TEST_ID);
}

test.describe("Authentication flow", () => {
  test("register, verify dashboard, sign out, login again", async ({
    page,
  }) => {
    const testUser = {
      displayName: "E2E Test User",
      email: `e2e-${Date.now()}-${Math.random().toString(36).substring(2, 8)}@example.com`,
      password: "testpassword123",
    };
    page.setDefaultTimeout(DEFAULT_TIMEOUT_MS);
    page.setDefaultNavigationTimeout(NAVIGATION_TIMEOUT_MS);

    // --- Register ---
    await openSignupForm(page);
    await page.locator('input#name').fill(testUser.displayName);
    await page.locator('input#email').fill(testUser.email);
    await page.locator('input#password').fill(testUser.password);
    await page.locator('button[type="submit"]').click();

    // Should land on dashboard
    await expect(page).toHaveURL("/");
    await expect(page.getByText("Get started with Biblioteka")).toBeVisible();
    await expect(page.getByText(testUser.email)).toBeVisible();

    // --- Sign out ---
    await page.locator('button#logout-button').click();
    await page.waitForSelector('button#login-btn', { timeout: NAVIGATION_TIMEOUT_MS });
    // Verify that the UI reflects a logged-out state and that auth token was cleared.
    await page.waitForSelector('input#email');
    await page.waitForSelector('input#password');
    await page.waitForFunction(() => localStorage.getItem('biblioteka_token') === null);

    // Should be on main page
    await expect(page).toHaveURL("/");
    await page.waitForSelector('button#login-btn', { timeout: NAVIGATION_TIMEOUT_MS });

    // --- Login ---
    await page.locator('button#login-btn').click();
    await page.waitForSelector('input#email');
    await page.waitForSelector('input#password');
    await page.locator('input#email').fill(testUser.email);
    await page.locator('input#password').fill(testUser.password);
    await page.locator('button[type="submit"]').click();

    // Should land on dashboard again
    await expect(page).toHaveURL("/");
    await expect(page.getByText("Get started with Biblioteka")).toBeVisible();
    await expect(page.getByText(testUser.email)).toBeVisible();
  });
});
