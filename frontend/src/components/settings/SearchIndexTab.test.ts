import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
  rebuildSearchIndex: vi
    .fn()
    .mockResolvedValue({ message: "search index rebuild started" }),
}));

vi.mock("lucide-svelte", () => ({
  Search: () => {},
}));

import SearchIndexTab from "./SearchIndexTab.svelte";
import { rebuildSearchIndex } from "../../lib/api";

describe("SearchIndexTab", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the Search Index heading", () => {
    render(SearchIndexTab);

    expect(
      screen.getByRole("heading", { name: /Search Index/i }),
    ).toBeInTheDocument();
  });

  it("renders the Rebuild Search Index button", () => {
    render(SearchIndexTab);

    expect(
      screen.getByRole("button", { name: "Rebuild Search Index" }),
    ).toBeInTheDocument();
  });

  it("calls rebuildSearchIndex when button is clicked", async () => {
    render(SearchIndexTab);

    const button = screen.getByRole("button", { name: "Rebuild Search Index" });
    await fireEvent.click(button);
    await tick();
    await tick();

    expect(rebuildSearchIndex).toHaveBeenCalledTimes(1);
  });

  it("shows success message after rebuild", async () => {
    render(SearchIndexTab);

    const button = screen.getByRole("button", { name: "Rebuild Search Index" });
    await fireEvent.click(button);
    await tick();
    await tick();

    expect(screen.getByRole("status")).toHaveTextContent(
      "search index rebuild started",
    );
  });

  it("shows error message on failure", async () => {
    vi.mocked(rebuildSearchIndex).mockRejectedValueOnce(
      new Error("server error"),
    );

    render(SearchIndexTab);

    const button = screen.getByRole("button", { name: "Rebuild Search Index" });
    await fireEvent.click(button);
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("server error");
  });

  it("disables the button while loading", async () => {
    let resolveRebuild!: (v: { message: string }) => void;
    vi.mocked(rebuildSearchIndex).mockReturnValueOnce(
      new Promise((resolve) => {
        resolveRebuild = resolve;
      }),
    );

    render(SearchIndexTab);

    const button = screen.getByRole("button", { name: "Rebuild Search Index" });
    await fireEvent.click(button);
    await tick();

    expect(screen.getByRole("button", { name: "Rebuilding…" })).toBeDisabled();

    resolveRebuild({ message: "done" });
    await tick();
    await tick();

    expect(
      screen.getByRole("button", { name: "Rebuild Search Index" }),
    ).toBeEnabled();
  });
});
