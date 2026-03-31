import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
  listKoboTokens: vi.fn().mockResolvedValue([
    {
      id: "tok-1",
      name: "My Kobo Libra",
      created_at: "2026-01-01T00:00:00Z",
    },
    {
      id: "tok-2",
      name: "Kobo Elipsa",
      created_at: "2026-02-01T00:00:00Z",
    },
  ]),
  createKoboToken: vi.fn(),
  deleteKoboToken: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("../../lib/clipboard", () => ({
  copyToClipboard: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("lucide-svelte", () => ({
  BookOpen: () => {},
  Copy: () => {},
  Trash2: () => {},
}));

import KoboTab from "./KoboTab.svelte";

describe("KoboTab delete confirmation", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("does not call deleteKoboToken when Delete button is clicked (shows confirmation instead)", async () => {
    const { deleteKoboToken } = await import("../../lib/api");
    render(KoboTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete token My Kobo Libra/,
    });
    await fireEvent.click(deleteButton);
    await tick();

    expect(deleteKoboToken).not.toHaveBeenCalled();
  });

  it("shows inline confirmation dialog when Delete is clicked", async () => {
    render(KoboTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete token My Kobo Libra/,
    });
    await fireEvent.click(deleteButton);
    await tick();

    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(screen.getByText(/Delete "My Kobo Libra"\?/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("dismisses confirmation dialog when Cancel is clicked", async () => {
    render(KoboTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete token My Kobo Libra/,
    });
    await fireEvent.click(deleteButton);
    await tick();

    const cancelButton = screen.getByRole("button", { name: "Cancel" });
    await fireEvent.click(cancelButton);
    await tick();

    expect(screen.queryByRole("alertdialog")).toBeNull();
  });

  it("calls deleteKoboToken after confirming deletion", async () => {
    const { deleteKoboToken } = await import("../../lib/api");
    vi.mocked(deleteKoboToken).mockClear();

    render(KoboTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete token My Kobo Libra/,
    });
    await fireEvent.click(deleteButton);
    await tick();

    const confirmButton = screen.getByRole("button", { name: "Delete" });
    await fireEvent.click(confirmButton);
    await tick();

    expect(deleteKoboToken).toHaveBeenCalledWith("tok-1");
  });

  it("only shows confirmation for the clicked token, not all tokens", async () => {
    render(KoboTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete token My Kobo Libra/,
    });
    await fireEvent.click(deleteButton);
    await tick();

    // Only one alertdialog should be shown
    expect(screen.getAllByRole("alertdialog")).toHaveLength(1);
    // The other delete button should still be visible
    expect(
      screen.getByRole("button", { name: /Delete token Kobo Elipsa/ }),
    ).toBeInTheDocument();
  });

  it("dismisses confirmation dialog when Escape is pressed", async () => {
    render(KoboTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete token My Kobo Libra/,
    });
    await fireEvent.click(deleteButton);
    await tick();

    expect(screen.getByRole("alertdialog")).toBeInTheDocument();

    const dialog = screen.getByRole("alertdialog");
    await fireEvent.keyDown(dialog, { key: "Escape" });
    await tick();

    expect(screen.queryByRole("alertdialog")).toBeNull();
  });

  it("moves focus to the Delete confirm button when dialog opens", async () => {
    render(KoboTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete token My Kobo Libra/,
    });
    await fireEvent.click(deleteButton);
    await tick();
    await tick();

    const confirmButton = screen.getByRole("button", { name: "Delete" });
    expect(document.activeElement).toBe(confirmButton);
  });
});
