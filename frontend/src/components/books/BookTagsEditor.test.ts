import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import { userEvent } from "@testing-library/user-event";
import type { Tag } from "../../types";

vi.mock("lucide-svelte", () => ({
  X: () => {},
  Tag: () => {},
  Plus: () => {},
  Loader: () => {},
}));

const mockGetBookTags = vi.fn();
const mockListTags = vi.fn();
const mockSetBookTags = vi.fn();
const mockCreateTag = vi.fn();

vi.mock("../../lib/api", () => ({
  getBookTags: (...args: unknown[]) => mockGetBookTags(...args),
  listTags: (...args: unknown[]) => mockListTags(...args),
  setBookTags: (...args: unknown[]) => mockSetBookTags(...args),
  createTag: (...args: unknown[]) => mockCreateTag(...args),
}));

import BookTagsEditor from "./BookTagsEditor.svelte";

const fakeTag1: Tag = {
  id: "t1",
  name: "fiction",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};
const fakeTag2: Tag = {
  id: "t2",
  name: "fantasy",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};
const fakeTag3: Tag = {
  id: "t3",
  name: "adventure",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("BookTagsEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => cleanup());

  it("shows loading state initially", () => {
    mockGetBookTags.mockReturnValue(new Promise(() => {}));
    mockListTags.mockReturnValue(new Promise(() => {}));
    render(BookTagsEditor, { bookId: "b1" });
    expect(screen.getByRole("status")).toHaveTextContent("Loading tags");
  });

  it("renders assigned tags as chips after loading", async () => {
    mockGetBookTags.mockResolvedValue([fakeTag1, fakeTag2]);
    mockListTags.mockResolvedValue([fakeTag1, fakeTag2, fakeTag3]);
    render(BookTagsEditor, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByText("fiction")).toBeInTheDocument();
      expect(screen.getByText("fantasy")).toBeInTheDocument();
    });
  });

  it("shows 'No tags assigned' when book has no tags", async () => {
    mockGetBookTags.mockResolvedValue([]);
    mockListTags.mockResolvedValue([fakeTag1]);
    render(BookTagsEditor, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByText("No tags assigned")).toBeInTheDocument();
    });
  });

  it("shows error message when loading fails", async () => {
    mockGetBookTags.mockRejectedValue(new Error("Load failed"));
    mockListTags.mockResolvedValue([]);
    render(BookTagsEditor, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByText("Load failed")).toBeInTheDocument();
    });
  });

  it("renders remove button for each assigned tag", async () => {
    mockGetBookTags.mockResolvedValue([fakeTag1]);
    mockListTags.mockResolvedValue([fakeTag1]);
    render(BookTagsEditor, { bookId: "b1" });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Remove tag fiction" }),
      ).toBeInTheDocument();
    });
  });

  it("removes a tag when remove button is clicked", async () => {
    mockGetBookTags.mockResolvedValue([fakeTag1, fakeTag2]);
    mockListTags.mockResolvedValue([fakeTag1, fakeTag2, fakeTag3]);
    mockSetBookTags.mockResolvedValue([fakeTag2]);

    const user = userEvent.setup();
    render(BookTagsEditor, { bookId: "b1" });

    await waitFor(() => {
      expect(screen.getByText("fiction")).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("button", { name: "Remove tag fiction" }),
    );

    await waitFor(() => {
      expect(mockSetBookTags).toHaveBeenCalledWith("b1", ["t2"]);
    });
  });

  it("opens dropdown when input is focused", async () => {
    mockGetBookTags.mockResolvedValue([]);
    mockListTags.mockResolvedValue([fakeTag1, fakeTag2]);

    const user = userEvent.setup();
    render(BookTagsEditor, { bookId: "b1" });

    await waitFor(() => {
      expect(
        screen.getByRole("combobox", { name: "Search or add tags" }),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("combobox", { name: "Search or add tags" }),
    );

    await waitFor(() => {
      expect(screen.getByRole("listbox")).toBeInTheDocument();
    });
  });

  it("filters tags based on search text", async () => {
    mockGetBookTags.mockResolvedValue([]);
    mockListTags.mockResolvedValue([fakeTag1, fakeTag2, fakeTag3]);

    const user = userEvent.setup();
    render(BookTagsEditor, { bookId: "b1" });

    await waitFor(() => {
      expect(
        screen.getByRole("combobox", { name: "Search or add tags" }),
      ).toBeInTheDocument();
    });

    await user.type(
      screen.getByRole("combobox", { name: "Search or add tags" }),
      "fan",
    );

    await waitFor(() => {
      expect(screen.getByRole("listbox")).toBeInTheDocument();
      expect(screen.getByText("fantasy")).toBeInTheDocument();
    });

    expect(screen.queryByText("fiction")).not.toBeInTheDocument();
    expect(screen.queryByText("adventure")).not.toBeInTheDocument();
  });

  it("adds a tag when clicked from dropdown", async () => {
    mockGetBookTags.mockResolvedValue([]);
    mockListTags.mockResolvedValue([fakeTag1, fakeTag2]);
    mockSetBookTags.mockResolvedValue([fakeTag1]);

    const user = userEvent.setup();
    render(BookTagsEditor, { bookId: "b1" });

    await waitFor(() => {
      expect(
        screen.getByRole("combobox", { name: "Search or add tags" }),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("combobox", { name: "Search or add tags" }),
    );
    await waitFor(() => screen.getByRole("listbox"));
    await user.click(screen.getByRole("option", { name: "fiction" }));

    await waitFor(() => {
      expect(mockSetBookTags).toHaveBeenCalledWith("b1", ["t1"]);
    });
  });

  it("shows create option when search text has no exact match", async () => {
    mockGetBookTags.mockResolvedValue([]);
    mockListTags.mockResolvedValue([fakeTag1]);

    const user = userEvent.setup();
    render(BookTagsEditor, { bookId: "b1" });

    await waitFor(() => {
      expect(
        screen.getByRole("combobox", { name: "Search or add tags" }),
      ).toBeInTheDocument();
    });

    await user.type(
      screen.getByRole("combobox", { name: "Search or add tags" }),
      "newgenre",
    );

    await waitFor(() => {
      expect(screen.getByText('Create "newgenre"')).toBeInTheDocument();
    });
  });

  it("does not show create option when search matches an existing tag exactly", async () => {
    mockGetBookTags.mockResolvedValue([]);
    mockListTags.mockResolvedValue([fakeTag1]);

    const user = userEvent.setup();
    render(BookTagsEditor, { bookId: "b1" });

    await waitFor(() => {
      expect(
        screen.getByRole("combobox", { name: "Search or add tags" }),
      ).toBeInTheDocument();
    });

    await user.type(
      screen.getByRole("combobox", { name: "Search or add tags" }),
      "fiction",
    );

    await waitFor(() => {
      expect(screen.queryByText('Create "fiction"')).not.toBeInTheDocument();
    });
  });

  it("creates and adds a new tag when create option is clicked", async () => {
    const newTag: Tag = {
      id: "t4",
      name: "newgenre",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    mockGetBookTags.mockResolvedValue([]);
    mockListTags.mockResolvedValue([fakeTag1]);
    mockCreateTag.mockResolvedValue(newTag);
    mockSetBookTags.mockResolvedValue([newTag]);

    const user = userEvent.setup();
    render(BookTagsEditor, { bookId: "b1" });

    await waitFor(() => {
      expect(
        screen.getByRole("combobox", { name: "Search or add tags" }),
      ).toBeInTheDocument();
    });

    await user.type(
      screen.getByRole("combobox", { name: "Search or add tags" }),
      "newgenre",
    );
    await waitFor(() => screen.getByText('Create "newgenre"'));
    await user.click(screen.getByText('Create "newgenre"'));

    await waitFor(() => {
      expect(mockCreateTag).toHaveBeenCalledWith({ name: "newgenre" });
      expect(mockSetBookTags).toHaveBeenCalledWith("b1", ["t4"]);
    });
  });
});
