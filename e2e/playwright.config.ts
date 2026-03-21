import { defineConfig } from "@playwright/test";
import { BASE_URL, TEST_PORT } from "./constants";

export default defineConfig({
  testDir: "./tests",
  globalSetup: "./global-setup.ts",
  fullyParallel: false,
  workers: 1,
  retries: 1,
  timeout: 30_000,
  expect: { timeout: 10_000 },

  use: {
    baseURL: BASE_URL,
    screenshot: "only-on-failure",
    trace: "on-first-retry",
  },

  projects: [
    {
      name: "chromium",
      use: { browserName: "chromium" },
    },
  ],

  webServer: {
    command: `../biblioteka -mode server`,
    url: `${BASE_URL}/api/health`,
    reuseExistingServer: !process.env.CI,
    timeout: 15_000,
    env: {
      PORT: String(TEST_PORT),
      JWT_SECRET: "e2e-test-jwt-secret",
    },
  },
});
