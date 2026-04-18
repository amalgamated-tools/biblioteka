import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/svelte";
import PasskeyList from "./PasskeyList.svelte";

vi.mock("lucide-svelte", () => ({
  KeyRound: () => {},
  Trash2: () => {},
}));

describe("PasskeyList contrast classes", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("uses higher-contrast classes for passkey text metadata", () => {
    vi.spyOn(Date.prototype, "toLocaleDateString").mockReturnValue("1/15/2026");

    render(PasskeyList, {
      props: {
        passkeys: [
          {
            id: "pk-1",
            name: "Laptop",
            aaguid: "aaguid-1",
            created_at: "2026-01-15T12:00:00Z",
          },
        ],
        passkeyDeleting: null,
        onDelete: vi.fn(),
      },
    });

    const createdDate = screen.getByText("1/15/2026");
    expect(createdDate).toHaveClass("text-ink-500");
    expect(createdDate).toHaveClass("dark:text-ink-300");
  });

  it("uses higher-contrast classes for the empty-state message", () => {
    render(PasskeyList, {
      props: {
        passkeys: [],
        passkeyDeleting: null,
        onDelete: vi.fn(),
      },
    });

    const emptyState = screen.getByText("No passkeys registered yet.");
    expect(emptyState).toHaveClass("text-ink-500");
    expect(emptyState).toHaveClass("dark:text-ink-300");
  });
});
