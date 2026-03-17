import { expect, type Page } from "@playwright/test";

export const AUTH_ERROR_TEST_ID = "auth-error";
export const DEFAULT_TIMEOUT_MS = Number(process.env.E2E_TIMEOUT_MS || 5000);
export const NAVIGATION_TIMEOUT_MS = Number(
  process.env.E2E_NAVIGATION_TIMEOUT_MS || 5000,
);

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
  await page.getByRole("button", { name: "Create Account" }).waitFor();
}

export async function signUp(page: Page, user: TestUser): Promise<void> {
  page.setDefaultTimeout(DEFAULT_TIMEOUT_MS);
  page.setDefaultNavigationTimeout(NAVIGATION_TIMEOUT_MS);

  await openSignupForm(page);
  await page.locator("input#name").fill(user.displayName);
  await page.locator("input#email").fill(user.email);
  await page.locator("input#password").fill(user.password);
  await page.locator("button[type='submit']").click();

  await expect(page).toHaveURL("/");
  await expect(page.getByText("Get started with Biblioteka")).toBeVisible();
  await expect(page.getByText(user.email)).toBeVisible();
}

export async function signOut(page: Page): Promise<void> {
  await page.locator("button#logout-button").click();
  await page.waitForSelector("button#login-btn", { timeout: NAVIGATION_TIMEOUT_MS });
  await page.waitForSelector("input#email");
  await page.waitForSelector("input#password");
  await page.waitForFunction(() => localStorage.getItem("biblioteka_token") === null);
  await expect(page.getByRole("button", { name: "Login", exact: true })).toBeVisible();
}

export async function signIn(page: Page, email: string, password: string): Promise<void> {
  await openAuthPage(page);
  await page.locator("input#email").fill(email);
  await page.locator("input#password").fill(password);
  await page.locator("button[type='submit']").click();
}
