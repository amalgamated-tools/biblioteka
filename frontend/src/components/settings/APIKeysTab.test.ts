import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
  listAPIKeys: vi.fn().mockResolvedValue([
    {
      id: "key-1",
      name: "CI Pipeline",
      key_prefix: "abc123",
      last_used_at: null,
      created_at: "2026-01-01T00:00:00Z",
    },
    {
      id: "key-2",
      name: "My Script",
      key_prefix: "def456",
      last_used_at: "2026-02-01T00:00:00Z",
      created_at: "2026-01-15T00:00:00Z",
    },
  ]),
  createAPIKey: vi.fn(),
  deleteAPIKey: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("../../lib/clipboard", () => ({
  copyToClipboard: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("lucide-svelte", () => ({
  KeyRound: () => {},
  Copy: () => {},
  Trash2: () => {},
}));

import APIKeysTab from "./APIKeysTab.svelte";

describe("APIKeysTab delete confirmation", () => {
  afterEach(() => {
    cleanup();
  });

  it("does not call deleteAPIKey when Delete button is clicked (shows confirmation instead)", async () => {
    const { deleteAPIKey } = await import("../../lib/api");
    render(APIKeysTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete API key CI Pipeline/,
    });
    await fireEvent.click(deleteButton);
    await tick();

    expect(deleteAPIKey).not.toHaveBeenCalled();
  });

  it("shows inline confirmation dialog when Delete is clicked", async () => {
    render(APIKeysTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete API key CI Pipeline/,
    });
    await fireEvent.click(deleteButton);
    await tick();

    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(screen.getByText(/Delete "CI Pipeline"\?/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("dismisses confirmation dialog when Cancel is clicked", async () => {
    render(APIKeysTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete API key CI Pipeline/,
    });
    await fireEvent.click(deleteButton);
    await tick();

    const cancelButton = screen.getByRole("button", { name: "Cancel" });
    await fireEvent.click(cancelButton);
    await tick();

    expect(screen.queryByRole("alertdialog")).toBeNull();
  });

  it("calls deleteAPIKey after confirming deletion", async () => {
    const { deleteAPIKey } = await import("../../lib/api");
    vi.mocked(deleteAPIKey).mockClear();

    render(APIKeysTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete API key CI Pipeline/,
    });
    await fireEvent.click(deleteButton);
    await tick();

    const confirmButton = screen.getByRole("button", { name: "Delete" });
    await fireEvent.click(confirmButton);
    await tick();

    expect(deleteAPIKey).toHaveBeenCalledWith("key-1");
  });

  it("only shows confirmation for the clicked key, not all keys", async () => {
    render(APIKeysTab);
    await tick();
    await tick();

    const deleteButton = screen.getByRole("button", {
      name: /Delete API key CI Pipeline/,
    });
    await fireEvent.click(deleteButton);
    await tick();

    // Only one alertdialog should be shown
    expect(screen.getAllByRole("alertdialog")).toHaveLength(1);
    // The other delete button should still be visible
    expect(
      screen.getByRole("button", { name: /Delete API key My Script/ }),
    ).toBeInTheDocument();
  });
});
