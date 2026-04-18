import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import { userEvent } from "@testing-library/user-event";

vi.mock("lucide-svelte", () => ({
  Tag: () => {},
  Plus: () => {},
  Check: () => {},
  X: () => {},
  Pencil: () => {},
  Trash2: () => {},
}));

vi.mock("../lib/actions", () => ({
  autofocusFirstButton: () => ({ destroy: () => {} }),
}));

const mockTagStore = {
  tags: [] as { id: string; name: string; created_at: string }[],
  loading: false,
  loaded: true,
  error: null as string | null,
  load: vi.fn(),
  add: vi.fn(),
  edit: vi.fn(),
  remove: vi.fn(),
};

vi.mock("../stores/tags.svelte", () => ({
  get tagStore() {
    return mockTagStore;
  },
}));

import Tags from "./Tags.svelte";

const fakeTags = [
  { id: "t1", name: "fiction", created_at: "2026-01-01T00:00:00Z" },
  { id: "t2", name: "fantasy", created_at: "2026-02-01T00:00:00Z" },
];

describe("Tags management page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockTagStore.tags = [];
    mockTagStore.loading = false;
    mockTagStore.loaded = true;
    mockTagStore.error = null;
  });

  afterEach(() => cleanup());

  it("renders Tags heading", () => {
    render(Tags);
    expect(
      screen.getByRole("heading", { name: "Tags", level: 1 }),
    ).toBeInTheDocument();
  });

  it("renders New Tag button", () => {
    render(Tags);
    expect(
      screen.getByRole("button", { name: /new tag/i }),
    ).toBeInTheDocument();
  });

  it("shows loading state", () => {
    mockTagStore.loading = true;
    mockTagStore.loaded = false;
    render(Tags);
    expect(screen.getByRole("status")).toHaveTextContent(/loading tags/i);
  });

  it("shows empty state when no tags exist", () => {
    render(Tags);
    expect(screen.getByText("No tags yet")).toBeInTheDocument();
  });

  it("renders tags in a table", () => {
    mockTagStore.tags = fakeTags;
    render(Tags);
    expect(screen.getByRole("table", { name: "Tags" })).toBeInTheDocument();
    expect(screen.getByText("fiction")).toBeInTheDocument();
    expect(screen.getByText("fantasy")).toBeInTheDocument();
  });

  it("shows create form when New Tag is clicked", async () => {
    const user = userEvent.setup();
    render(Tags);
    await user.click(screen.getByRole("button", { name: /new tag/i }));
    expect(
      screen.getByRole("heading", { name: "Create Tag", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: /name/i })).toBeInTheDocument();
  });

  it("calls tagStore.add when form is submitted", async () => {
    mockTagStore.add.mockResolvedValue({
      id: "t3",
      name: "horror",
      created_at: "2026-01-01T00:00:00Z",
    });
    const user = userEvent.setup();
    render(Tags);

    await user.click(screen.getByRole("button", { name: /new tag/i }));
    await user.type(screen.getByLabelText(/^Name/), "horror");
    await user.click(screen.getByRole("button", { name: /^Create/ }));

    await waitFor(() => {
      expect(mockTagStore.add).toHaveBeenCalledWith({ name: "horror" });
    });
  });

  it("shows inline rename form when Rename is clicked", async () => {
    mockTagStore.tags = fakeTags;
    const user = userEvent.setup();
    render(Tags);

    await user.click(
      screen.getByRole("button", { name: "Rename tag fiction" }),
    );

    expect(
      screen.getByRole("textbox", { name: "Tag name" }),
    ).toBeInTheDocument();
  });

  it("calls tagStore.edit when rename is submitted", async () => {
    mockTagStore.tags = fakeTags;
    mockTagStore.edit.mockResolvedValue({
      id: "t1",
      name: "literary fiction",
      created_at: "2026-01-01T00:00:00Z",
    });
    const user = userEvent.setup();
    render(Tags);

    await user.click(
      screen.getByRole("button", { name: "Rename tag fiction" }),
    );
    const input = screen.getByRole("textbox", { name: "Tag name" });
    await user.clear(input);
    await user.type(input, "literary fiction");
    await user.click(screen.getByRole("button", { name: "Save tag name" }));

    await waitFor(() => {
      expect(mockTagStore.edit).toHaveBeenCalledWith("t1", {
        name: "literary fiction",
      });
    });
  });

  it("shows delete confirmation when Delete is clicked", async () => {
    mockTagStore.tags = fakeTags;
    const user = userEvent.setup();
    render(Tags);

    await user.click(
      screen.getByRole("button", { name: "Delete tag fiction" }),
    );

    expect(screen.getByText(/Delete "fiction"\?/)).toBeInTheDocument();
  });

  it("calls tagStore.remove when delete is confirmed", async () => {
    mockTagStore.tags = fakeTags;
    mockTagStore.remove.mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(Tags);

    await user.click(
      screen.getByRole("button", { name: "Delete tag fiction" }),
    );
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(mockTagStore.remove).toHaveBeenCalledWith("t1");
    });
  });

  it("cancels delete when Cancel is clicked", async () => {
    mockTagStore.tags = fakeTags;
    const user = userEvent.setup();
    render(Tags);

    await user.click(
      screen.getByRole("button", { name: "Delete tag fiction" }),
    );
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(screen.queryByText(/Delete "fiction"\?/)).not.toBeInTheDocument();
    expect(mockTagStore.remove).not.toHaveBeenCalled();
  });

  it("shows error from store", () => {
    mockTagStore.error = "Failed to load tags";
    mockTagStore.loaded = false;
    render(Tags);
    expect(screen.getByText("Failed to load tags")).toBeInTheDocument();
  });
});
