import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/svelte";

const { listAPIKeysMock } = vi.hoisted(() => ({
  listAPIKeysMock: vi.fn().mockRejectedValue(new Error("Load failed")),
}));

vi.mock("../../lib/api", () => ({
  listAPIKeys: listAPIKeysMock,
  createAPIKey: vi.fn(),
  deleteAPIKey: vi.fn(),
}));

vi.mock("lucide-svelte", () => ({
  KeyRound: () => {},
  Copy: () => {},
  Trash2: () => {},
}));

import APIKeysTab from "./APIKeysTab.svelte";

describe("APIKeysTab accessibility", () => {
  it("announces API key loading failures as assertive alerts", async () => {
    render(APIKeysTab);

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("Load failed");
    });

    expect(screen.getByRole("alert")).toHaveAttribute("aria-live", "assertive");
  });
});
