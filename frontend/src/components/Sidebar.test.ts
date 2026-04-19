import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";

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
  BookMarked: () => {},
  Settings2: () => {},
  Users: () => {},
  Tag: () => {},
}));

import Sidebar from "./Sidebar.svelte";

const originalInnerWidth = window.innerWidth;

function setViewportWidth(width: number) {
  Object.defineProperty(window, "innerWidth", {
    configurable: true,
    writable: true,
    value: width,
  });
  window.dispatchEvent(new Event("resize"));
}

describe("Sidebar navigation accessibility", () => {
  afterEach(() => {
    cleanup();
    Object.defineProperty(window, "innerWidth", {
      configurable: true,
      writable: true,
      value: originalInnerWidth,
    });
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

    const groups = screen.getByRole("link", { name: "Reading Groups" });
    expect(groups).toHaveAttribute("href", "#groups");
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

  it("sets aria-current='page' on the Reading Groups link when active", () => {
    render(Sidebar, {
      props: { currentView: "groups", open: true, onClose: () => {} },
    });

    const groups = screen.getByRole("link", { name: "Reading Groups" });
    expect(groups).toHaveAttribute("aria-current", "page");

    const dashboard = screen.getByRole("link", { name: "Dashboard" });
    expect(dashboard).not.toHaveAttribute("aria-current");
  });

  it("renders Tags link with correct href", () => {
    render(Sidebar, {
      props: { currentView: "dashboard", open: true, onClose: () => {} },
    });

    const tags = screen.getByRole("link", { name: "Tags" });
    expect(tags).toHaveAttribute("href", "#tags");
  });

  it("sets aria-current='page' on the Tags link when active", () => {
    render(Sidebar, {
      props: { currentView: "tags", open: true, onClose: () => {} },
    });

    const tags = screen.getByRole("link", { name: "Tags" });
    expect(tags).toHaveAttribute("aria-current", "page");

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

  it("library settings links include the library name in aria-label (WCAG 2.4.6)", () => {
    render(Sidebar, {
      props: { currentView: "dashboard", open: true, onClose: () => {} },
    });

    const fictionSettings = screen.getByRole("link", {
      name: "Library settings for Fiction",
    });
    expect(fictionSettings).toBeInTheDocument();
    expect(fictionSettings).toHaveAttribute("href", "#libraries/edit/1");

    const nonFictionSettings = screen.getByRole("link", {
      name: "Library settings for Non-Fiction",
    });
    expect(nonFictionSettings).toBeInTheDocument();
    expect(nonFictionSettings).toHaveAttribute("href", "#libraries/edit/2");
  });

  it("library settings links use a contrast-safe default color (WCAG 1.4.11)", () => {
    render(Sidebar, {
      props: { currentView: "dashboard", open: true, onClose: () => {} },
    });

    const fictionSettings = screen.getByRole("link", {
      name: "Library settings for Fiction",
    });
    expect(fictionSettings).toHaveClass("text-ink-500");
    expect(fictionSettings).not.toHaveClass("text-ink-600");
    expect(fictionSettings.className).toContain("group-hover:text-accent-400");
    expect(fictionSettings.className).toContain(
      "group-focus-within:text-accent-400",
    );
    expect(fictionSettings).not.toHaveClass("opacity-0");
    expect(fictionSettings).not.toHaveClass("opacity-30");
    expect(fictionSettings).not.toHaveClass("opacity-100");
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

  it("uses higher-contrast classes for sidebar metadata text", () => {
    render(Sidebar, {
      props: { currentView: "dashboard", open: true, onClose: () => {} },
    });

    const userMeta = screen.getByText("test@example.com");
    expect(userMeta).toHaveClass("text-ink-300");

    const homeHeading = screen.getByRole("heading", { name: "Home", level: 2 });
    expect(homeHeading).toHaveClass("text-ink-300");

    const librariesHeading = screen.getByRole("heading", {
      name: "Libraries",
      level: 2,
    });
    expect(librariesHeading).toHaveClass("text-ink-300");

    const logout = screen.getByRole("button", { name: "Logout" });
    expect(logout).toHaveClass("text-ink-300");
  });

  it("does not render the app name as a heading", () => {
    render(Sidebar, {
      props: { currentView: "dashboard", open: true, onClose: () => {} },
    });

    expect(
      screen.queryByRole("heading", { name: "biblioteka" }),
    ).not.toBeInTheDocument();
  });

  it("backdrop button does not have tabindex=-1 so keyboard users can close the sidebar", () => {
    render(Sidebar, {
      props: { currentView: "dashboard", open: true, onClose: () => {} },
    });

    const backdrop = screen.getByRole("button", { name: "Close sidebar" });
    expect(backdrop).not.toHaveAttribute("tabindex");
  });

  it("calls onClose when Escape key is pressed while sidebar is open", async () => {
    const onClose = vi.fn();
    render(Sidebar, {
      props: { currentView: "dashboard", open: true, onClose },
    });

    await fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not call onClose when Escape key is pressed while sidebar is closed", async () => {
    const onClose = vi.fn();
    render(Sidebar, {
      props: { currentView: "dashboard", open: false, onClose },
    });

    await fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
  });

  it("hides the sidebar from assistive technology when closed on mobile", () => {
    setViewportWidth(375);
    const { container } = render(Sidebar, {
      props: { currentView: "dashboard", open: false, onClose: () => {} },
    });

    const aside = container.querySelector("aside") as HTMLElement;
    expect(aside).toHaveAttribute("id", "main-sidebar");
    expect(aside).toHaveAttribute("aria-hidden", "true");
    expect(aside.inert).toBe(true);
  });

  it("keeps the sidebar accessible on desktop even when closed", () => {
    setViewportWidth(1024);
    const { container } = render(Sidebar, {
      props: { currentView: "dashboard", open: false, onClose: () => {} },
    });

    const aside = container.querySelector("aside") as HTMLElement;
    expect(aside).toHaveAttribute("id", "main-sidebar");
    expect(aside).not.toHaveAttribute("aria-hidden");
    expect(aside.inert).toBe(false);
  });
});
