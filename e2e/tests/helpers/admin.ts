import type { Page } from "@playwright/test";
import { ADMIN_EMAIL, ADMIN_PASSWORD } from "../../constants";
import { openLoginForm, signIn } from "./auth";

export { ADMIN_EMAIL, ADMIN_PASSWORD };

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
