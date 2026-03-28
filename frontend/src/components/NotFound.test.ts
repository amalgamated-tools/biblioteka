import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    navigate: vi.fn(),
  },
}));

import { routerStore } from "../stores/router.svelte";
import NotFound from "./NotFound.svelte";

describe("NotFound", () => {
  afterEach(() => {
    cleanup();
    vi.mocked(routerStore.navigate).mockClear();
  });

  it("renders the 404 heading and description", () => {
    render(NotFound);

    expect(
      screen.getByRole("heading", { name: "Page Not Found" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("The page you are looking for does not exist."),
    ).toBeInTheDocument();
  });

  it("renders the decorative 404 text as aria-hidden", () => {
    const { container } = render(NotFound);

    const decorative = container.querySelector('[aria-hidden="true"]');
    expect(decorative).toBeInTheDocument();
    expect(decorative?.textContent?.trim()).toBe("404");
  });

  it("renders a 'Go to Dashboard' button", () => {
    render(NotFound);

    const button = screen.getByRole("button", { name: "Go to Dashboard" });
    expect(button).toBeInTheDocument();
  });

  it("navigates to dashboard when the button is clicked", async () => {
    render(NotFound);

    const button = screen.getByRole("button", { name: "Go to Dashboard" });
    await fireEvent.click(button);

    expect(routerStore.navigate).toHaveBeenCalledWith("dashboard");
  });
});
