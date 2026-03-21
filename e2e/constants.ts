/** Shared constants for E2E test configuration. */

export const TEST_PORT = 3847;
export const BASE_URL = `http://localhost:${TEST_PORT}`;

// Admin user credentials — the global setup creates this user as the very
// first user in the database, which makes them an admin automatically.
export const ADMIN_EMAIL = "e2e-admin@biblioteka-e2e.test";
export const ADMIN_PASSWORD = "adminpassword123";
export const ADMIN_NAME = "E2E Admin";
