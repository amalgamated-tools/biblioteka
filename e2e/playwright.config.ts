import { defineConfig } from "@playwright/test";

const testPort = 3847;

export default defineConfig({
  testDir: "./tests",
  fullyParallel: false,
  workers: 1,
  retries: 1,
  timeout: 30_000,
  expect: { timeout: 10_000 },

  use: {
    baseURL: `http://localhost:${testPort}`,
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
    url: `http://localhost:${testPort}/api/health`,
    reuseExistingServer: !process.env.CI,
    timeout: 15_000,
    env: {
      PORT: String(testPort),
      JWT_SECRET: "e2e-test-jwt-secret",
    },
  },
});
