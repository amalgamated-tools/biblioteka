import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";

vi.mock("../stores/auth.svelte", () => ({
  authStore: {
    user: { id: "1", email: "test@example.com" },
    signOut: vi.fn(),
  },
}));

vi.mock("../stores/libraries.svelte", () => ({
  libraryStore: {
    loaded: true,
    libraries: [
      { id: 1, name: "Fiction" },
      { id: 2, name: "Non-Fiction" },
    ],
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

describe("Sidebar navigation accessibility", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders Dashboard, All Books, and Settings as links with correct hrefs", () => {
    render(Sidebar, {
      props: { currentView: "dashboard", open: true, onClose: () => {} },
    });

    const dashboard = screen.getByRole("link", { name: "Dashboard" });
    expect(dashboard).toHaveAttribute("href", "#dashboard");

    const books = screen.getByRole("link", { name: "All Books" });
    expect(books).toHaveAttribute("href", "#books");

    const settings = screen.getByRole("link", { name: "Settings" });
    expect(settings).toHaveAttribute("href", "#settings");
  });

  it("renders Logout as a button, not a link", () => {
    render(Sidebar, {
      props: { currentView: "dashboard", open: true, onClose: () => {} },
    });

    const logout = screen.getByRole("button", { name: "Logout" });
    expect(logout).toBeInTheDocument();
  });

  it("sets aria-current='page' on the active navigation link", () => {
    render(Sidebar, {
      props: { currentView: "settings", open: true, onClose: () => {} },
    });

    const settings = screen.getByRole("link", { name: "Settings" });
    expect(settings).toHaveAttribute("aria-current", "page");

    const dashboard = screen.getByRole("link", { name: "Dashboard" });
    expect(dashboard).not.toHaveAttribute("aria-current");
  });

  it("sets aria-current='page' on the active library link", () => {
    render(Sidebar, {
      props: {
        currentView: "libraries",
        subPath: "1",
        open: true,
        onClose: () => {},
      },
    });

    const fiction = screen.getByRole("link", { name: "Fiction" });
    expect(fiction).toHaveAttribute("aria-current", "page");

    const nonFiction = screen.getByRole("link", { name: "Non-Fiction" });
    expect(nonFiction).not.toHaveAttribute("aria-current");
  });

  it("renders navigation group labels as headings", () => {
    render(Sidebar, {
      props: { currentView: "dashboard", open: true, onClose: () => {} },
    });

    expect(
      screen.getByRole("heading", { name: "Home", level: 2 }),
    ).toBeVisible();
    expect(
      screen.getByRole("heading", { name: "Libraries", level: 2 }),
    ).toBeVisible();
  });

  it("does not render the app name as a heading", () => {
    render(Sidebar, {
      props: { currentView: "dashboard", open: true, onClose: () => {} },
    });

    expect(
      screen.queryByRole("heading", { name: "biblioteka" }),
    ).not.toBeInTheDocument();
  });
});
