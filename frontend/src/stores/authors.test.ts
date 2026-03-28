import { describe, it, expect, beforeEach, vi } from "vitest";
import { authorStore } from "./authors.svelte";
import * as api from "../lib/api";
import type { Author } from "../types";

vi.mock("../lib/api", async () => {
  return {
    listAuthors: vi.fn(),
    createAuthor: vi.fn(),
    updateAuthor: vi.fn(),
    deleteAuthor: vi.fn(),
  };
});

const fakeAuthor: Author = {
  id: "a1",
  name: "Test Author",
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  image_url: null,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("author store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authorStore.authors = [];
    authorStore.loading = false;
    authorStore.loaded = false;
  });

  describe("load", () => {
    it("fetches authors and sets loaded", async () => {
      vi.mocked(api.listAuthors).mockResolvedValue([fakeAuthor]);

      await authorStore.load();

      expect(api.listAuthors).toHaveBeenCalledTimes(1);
      expect(authorStore.authors).toEqual([fakeAuthor]);
      expect(authorStore.loaded).toBe(true);
      expect(authorStore.loading).toBe(false);
    });

    it("does not call API again after already loaded", async () => {
      vi.mocked(api.listAuthors).mockResolvedValue([fakeAuthor]);

      await authorStore.load();
      await authorStore.load();

      expect(api.listAuthors).toHaveBeenCalledTimes(1);
    });

    it("calls API only once when invoked concurrently", async () => {
      vi.mocked(api.listAuthors).mockResolvedValue([fakeAuthor]);

      await Promise.all([
        authorStore.load(),
        authorStore.load(),
        authorStore.load(),
      ]);

      expect(api.listAuthors).toHaveBeenCalledTimes(1);
      expect(authorStore.authors).toEqual([fakeAuthor]);
      expect(authorStore.loaded).toBe(true);
    });

    it("resets loading on API error", async () => {
      vi.mocked(api.listAuthors).mockRejectedValue(new Error("fail"));

      await authorStore.load();

      expect(authorStore.loading).toBe(false);
      expect(authorStore.loaded).toBe(false);
      expect(authorStore.authors).toEqual([]);
    });
  });
});
