import { describe, expect, it, vi, afterEach } from "vitest";
import { tick } from "svelte";
import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import type { User } from "./types";

const authStoreMock = vi.hoisted(() => ({
  loading: false,
  user: {
    id: "1",
    email: "test@example.com",
    oidc_linked: false,
    is_admin: false,
  } as User | null,
  init: vi.fn(),
}));

vi.mock("./stores/auth.svelte", () => ({
  authStore: authStoreMock,
}));

vi.mock("./stores/router.svelte", () => ({
  routerStore: {
    currentView: "dashboard",
    hash: "dashboard",
    isKnownView: true,
    pageTitle: "Dashboard – biblioteka",
    navigate: vi.fn(),
  },
}));

vi.mock("./stores/libraries.svelte", () => ({
  libraryStore: {
    loaded: true,
    libraries: [],
  },
}));

vi.mock("./components/Auth.svelte", () => ({ default: () => {} }));
vi.mock("./components/Dashboard.svelte", () => ({ default: () => {} }));
vi.mock("./components/Books.svelte", () => ({ default: () => {} }));
vi.mock("./components/MyLibrary.svelte", () => ({ default: () => {} }));
vi.mock("./components/Libraries.svelte", () => ({ default: () => {} }));
vi.mock("./components/Sidebar.svelte");
vi.mock("./components/Settings.svelte", () => ({ default: () => {} }));
vi.mock("lucide-svelte", () => ({ Menu: () => {} }));

import App from "./App.svelte";

describe("App", () => {
  afterEach(() => {
    cleanup();
  });

  it("sets document.title from routerStore.pageTitle on mount", async () => {
    render(App);
    await tick();
    expect(document.title).toBe("Dashboard – biblioteka");
  });

  it("does not steal focus on initial mount", async () => {
    render(App);
    await tick();

    const main = screen.getByRole("main");
    // The focus effect skips the first run, so main should NOT be focused
    // on initial render — this avoids stealing focus from the browser UI
    // (e.g. address bar) on a hard refresh.
    expect(document.activeElement).not.toBe(main);
  });

  it("provides a functional skip link that moves focus to the main content", async () => {
    const user = userEvent.setup();
    const { container } = render(App);

    const skipLink = screen.getByRole("link", {
      name: /skip to main content/i,
    });

    // Ensure the main content region exists and is the target.
    const main = screen.getByRole("main");
    expect(main.id).toBe("main-content");

    // Ensure the skip link appears before the sidebar in DOM order.
    const sidebar =
      container.querySelector("aside") ??
      container.querySelector('[data-testid="sidebar"]');
    expect(sidebar).not.toBeNull();
    const position = skipLink.compareDocumentPosition(sidebar!);
    expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    );

    // Activating the skip link should move focus to the main content.
    skipLink.focus();
    expect(document.activeElement).toBe(skipLink);

    await user.click(skipLink);

    expect(document.activeElement).toBe(main);
  });

  it("sets inert on main and header when the mobile sidebar is open", async () => {
    const user = userEvent.setup();
    render(App);
    await tick();

    const main = screen.getByRole("main") as HTMLElement;
    const header = screen.getByRole("banner", {
      name: "Mobile header",
    }) as HTMLElement;

    // Svelte 5 sets the DOM property (not the HTML attribute), so we assert
    // on the property directly — jsdom does not reflect .inert back to an
    // attribute, making toHaveAttribute("inert") unreliable here.
    expect(main.inert).toBe(false);
    expect(header.inert).toBe(false);

    const openMenuButton = screen.getByRole("button", { name: "Open menu" });
    await user.click(openMenuButton);
    await tick();

    expect(main.inert).toBe(true);
    expect(header.inert).toBe(true);
  });

  it("hides the decorative spinner from screen readers and exposes the loading message as a status", async () => {
    authStoreMock.loading = true;
    authStoreMock.user = null;

    try {
      const { container } = render(App);
      await tick();

      // The decorative spinner container must be hidden from screen readers.
      const spinnerContainer = container.querySelector('[aria-hidden="true"]');
      expect(spinnerContainer).not.toBeNull();

      // The loading message must carry role="status" so assistive technology
      // announces it as a live status notification.
      const statusEl = screen.getByRole("status");
      expect(statusEl.textContent?.trim()).toBe("Loading your library…");
    } finally {
      authStoreMock.loading = false;
      authStoreMock.user = {
        id: "1",
        email: "test@example.com",
        oidc_linked: false,
        is_admin: false,
      };
    }
  });
});
