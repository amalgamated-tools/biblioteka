import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/svelte";
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
    if (sidebar) {
      const position = skipLink.compareDocumentPosition(sidebar);
      expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
        Node.DOCUMENT_POSITION_FOLLOWING,
      );
    }

    // Activating the skip link should move focus to the main content.
    skipLink.focus();
    expect(document.activeElement).toBe(skipLink);

    await fireEvent.click(skipLink);

    expect(document.activeElement).toBe(main);
  });
});
