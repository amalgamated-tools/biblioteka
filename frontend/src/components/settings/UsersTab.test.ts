import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";

vi.mock("../../stores/auth.svelte", () => ({
  authStore: {
    user: { id: "1", email: "admin@example.com" },
  },
}));

vi.mock("../../lib/api", () => ({
  listUsers: vi.fn().mockResolvedValue([]),
  setUserAdmin: vi.fn(),
}));

vi.mock("lucide-svelte", () => ({
  Users: () => {},
}));

import UsersTab from "./UsersTab.svelte";

describe("UsersTab accessibility", () => {
  afterEach(() => {
    cleanup();
  });

  it("marks each table header as a column header", () => {
    render(UsersTab, {
      props: {
        cachedUsers: [
          {
            id: "1",
            name: "Admin User",
            email: "admin@example.com",
            is_admin: true,
            oidc_linked: false,
            created_at: "2026-01-01T00:00:00Z",
          },
        ],
        onUsersLoaded: () => {},
      },
    });

    for (const name of ["Name", "Email", "Type", "Role", "Joined"]) {
      expect(screen.getByRole("columnheader", { name })).toHaveAttribute(
        "scope",
        "col",
      );
    }
  });
});
