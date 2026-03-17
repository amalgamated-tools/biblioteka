import { test, expect } from "@playwright/test";
import {
  createTestUser,
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
});
