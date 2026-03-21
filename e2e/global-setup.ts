import { request } from "@playwright/test";
import { ADMIN_EMAIL, ADMIN_NAME, ADMIN_PASSWORD, BASE_URL } from "./constants";

export default async function globalSetup() {
  const context = await request.newContext({ baseURL: BASE_URL });

  try {
    const res = await context.post("/api/auth/signup", {
      data: {
        name: ADMIN_NAME,
        email: ADMIN_EMAIL,
        password: ADMIN_PASSWORD,
      },
    });

    if (res.status() === 201) {
      const body = await res.json();
      if (!body.user?.is_admin) {
        throw new Error(
          "globalSetup: admin user was created but was NOT promoted to admin — " +
            "the database is not empty. Ensure a clean DB before running E2E tests.",
        );
      }
    } else if (res.status() !== 409) {
      // 409 = user already exists (dev-server reuse); any other error is fatal.
      throw new Error(
        `globalSetup: failed to create admin user: HTTP ${res.status()}: ${await res.text()}`,
      );
    }
  } finally {
    await context.dispose();
  }
}
