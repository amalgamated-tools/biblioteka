import { request } from "@playwright/test";

const testPort = 3847;

// These credentials must match the constants in tests/helpers/admin.ts.
// The admin user is the first user registered in the database.
// In CI the database is always empty at test-suite start so this signup
// creates the first (admin) user.  On a reused dev server a 409 response
// means the user already exists; we treat that as success and let the tests
// sign in normally.
export const ADMIN_EMAIL = "e2e-admin@biblioteka-e2e.test";
export const ADMIN_PASSWORD = "adminpassword123";
const ADMIN_NAME = "E2E Admin";

export default async function globalSetup() {
  const context = await request.newContext({
    baseURL: `http://localhost:${testPort}`,
  });

  try {
    const res = await context.post("/api/auth/signup", {
      data: {
        name: ADMIN_NAME,
        email: ADMIN_EMAIL,
        password: ADMIN_PASSWORD,
      },
    });

    // 201 = created (first user → auto-admin), 409 = already exists (dev reuse).
    if (!res.ok() && res.status() !== 409) {
      throw new Error(
        `globalSetup: failed to create admin user: HTTP ${res.status()}: ${await res.text()}`,
      );
    }
  } finally {
    await context.dispose();
  }
}
