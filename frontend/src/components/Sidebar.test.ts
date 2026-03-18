import { describe, expect, it, vi, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/svelte";

vi.mock("../stores/auth.svelte", () => ({
  authStore: {
    user: {
      id: "1",
      email: "test@example.com",
      oidc_linked: false,
      is_admin: false,
    },
  },
}));

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    navigate: vi.fn(),
  },
}));

vi.mock("../stores/libraries.svelte", () => ({
  libraryStore: {
    loaded: true,
    libraries: [{ id: "lib1", name: "My Library" }],
    load: vi.fn(),
  },
}));

vi.mock("../lib/api", () => ({
  getVersion: vi.fn().mockResolvedValue("1.0.0"),
}));

vi.mock("lucide-svelte", () => ({
  LayoutDashboard: () => {},
  BookOpen: () => {},
  Library: () => {},
  Plus: () => {},
  Settings: () => {},
  LogOut: () => {},
  BookCheck: () => {},
  Settings2: () => {},
}));

import Sidebar from "./Sidebar.svelte";

afterEach(() => {
  cleanup();
});

describe("Sidebar", () => {
  describe("aria-current on navigation buttons", () => {
    it("sets aria-current='page' only on Dashboard when it is the current view", () => {
      render(Sidebar, {
        currentView: "dashboard",
        onNavigate: vi.fn(),
        open: true,
        onClose: vi.fn(),
      });

      expect(
        screen.getByRole("button", { name: /dashboard/i }),
      ).toHaveAttribute("aria-current", "page");

      expect(
        screen.getByRole("button", { name: /all books/i }),
      ).not.toHaveAttribute("aria-current");

      expect(
        screen.getByRole("button", { name: /^settings$/i }),
      ).not.toHaveAttribute("aria-current");
    });

    it("sets aria-current='page' only on All Books when it is the current view", () => {
      render(Sidebar, {
        currentView: "books",
        onNavigate: vi.fn(),
        open: true,
        onClose: vi.fn(),
      });

      expect(
        screen.getByRole("button", { name: /dashboard/i }),
      ).not.toHaveAttribute("aria-current");

      expect(
        screen.getByRole("button", { name: /all books/i }),
      ).toHaveAttribute("aria-current", "page");

      expect(
        screen.getByRole("button", { name: /^settings$/i }),
      ).not.toHaveAttribute("aria-current");
    });

    it("sets aria-current='page' only on Settings when it is the current view", () => {
      render(Sidebar, {
        currentView: "settings",
        onNavigate: vi.fn(),
        open: true,
        onClose: vi.fn(),
      });

      expect(
        screen.getByRole("button", { name: /dashboard/i }),
      ).not.toHaveAttribute("aria-current");

      expect(
        screen.getByRole("button", { name: /all books/i }),
      ).not.toHaveAttribute("aria-current");

      expect(
        screen.getByRole("button", { name: /^settings$/i }),
      ).toHaveAttribute("aria-current", "page");
    });
  });
});
