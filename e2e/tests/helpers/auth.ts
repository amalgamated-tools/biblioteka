import { expect, type Page } from "@playwright/test";

const AUTH_ERROR_TEST_ID = "auth-error";
const TOKEN_KEY = "biblioteka_token";
const DEFAULT_TIMEOUT_MS = (() => {
  const v = parseInt(process.env.E2E_TIMEOUT_MS ?? "", 10);
  return Number.isFinite(v) ? v : 5000;
})();
const NAVIGATION_TIMEOUT_MS = (() => {
  const v = parseInt(process.env.E2E_NAVIGATION_TIMEOUT_MS ?? "", 10);
  return Number.isFinite(v) ? v : 5000;
})();

export interface TestUser {
  displayName: string;
  email: string;
  password: string;
}

export function createTestUser(overrides: Partial<TestUser> = {}): TestUser {
  const uniquePart = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

  return {
    displayName: "E2E Test User",
    email: `e2e-${uniquePart}@example.com`,
    password: "testpassword123",
    ...overrides,
  };
}

export function getAuthErrorBanner(page: Page) {
  return page.getByTestId(AUTH_ERROR_TEST_ID);
}

export function configureTimeouts(page: Page): void {
  page.setDefaultTimeout(DEFAULT_TIMEOUT_MS);
  page.setDefaultNavigationTimeout(NAVIGATION_TIMEOUT_MS);
}

export async function openAuthPage(page: Page): Promise<void> {
  await page.goto(`/`, {
    waitUntil: "networkidle",
    timeout: NAVIGATION_TIMEOUT_MS,
  });
  await page.waitForSelector("button#login-btn");
}

export async function openSignupForm(page: Page): Promise<void> {
  await openAuthPage(page);
  await page.getByRole("button", { name: "Sign Up", exact: true }).click();
  await page.waitForSelector("input#name");
  await page.waitForFunction(() => {
    const btn = document.querySelector('button[type="submit"]');
    return btn && btn.textContent?.trim() === "Create Account";
  });
}

export async function signUp(page: Page, user: TestUser): Promise<void> {
  await openSignupForm(page);
  await page.locator("input#name").fill(user.displayName);
  await page.locator("input#email").fill(user.email);
  await page.locator("input#password").fill(user.password);
  await page.locator("button[type='submit']").click();

  await expect(page).toHaveURL("/");
}

export async function signOut(page: Page): Promise<void> {
  await page.locator("button#logout-button").click();
  await page.waitForSelector("button#login-btn");
  await page.locator("button#login-btn").click();
  await page.waitForSelector("input#email");
  await page.waitForSelector("input#password");
  await page.waitForFunction((tokenKey) => localStorage.getItem(tokenKey) === null, TOKEN_KEY);
  await expect(page.getByRole("button", { name: "Login", exact: true })).toBeVisible();
}

export async function openLoginForm(page: Page): Promise<void> {
  await openAuthPage(page);
  await page.locator("button#login-btn").click();
  await page.waitForSelector("input#email");
}

// Callers assert the expected post-login outcome because some tests expect failure.
// Expects the login form to already be visible (via openLoginForm or signOut).
export async function signIn(page: Page, email: string, password: string): Promise<void> {
  await page.locator("input#email").fill(email);
  await page.locator("input#password").fill(password);
  await page.locator("button[type='submit']").click();
}
