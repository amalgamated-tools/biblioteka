import type { Page } from "@playwright/test";
import { openLoginForm, signIn } from "./auth";

// These credentials must match those in e2e/global-setup.ts.
// The global setup creates this user as the very first user in the database,
// which makes them an admin automatically.
export const ADMIN_EMAIL = "e2e-admin@biblioteka-e2e.test";
export const ADMIN_PASSWORD = "adminpassword123";

/**
 * Signs in as the pre-seeded admin user and waits until the authenticated
 * dashboard is fully loaded.
 */
export async function signInAsAdmin(page: Page): Promise<void> {
  await openLoginForm(page);
  await signIn(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  // Wait for the logout button which is only present in the authenticated view.
  await page.waitForSelector("button#logout-button");
}
