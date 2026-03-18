import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/svelte";

vi.mock("./stores/auth.svelte", () => ({
  authStore: {
    loading: false,
    user: { id: "1", email: "test@example.com", oidc_linked: false, is_admin: false },
    init: vi.fn(),
  },
}));

vi.mock("./stores/router.svelte", () => ({
  routerStore: {
    currentView: "dashboard",
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
  it("provides a functional skip link that moves focus to the main content", async () => {
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

    await fireEvent.click(skipLink);

    expect(document.activeElement).toBe(main);
  });
});
