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

  describe("add", () => {
    it("appends the created author to the store", async () => {
      vi.mocked(api.createAuthor).mockResolvedValue(fakeAuthor);

      const result = await authorStore.add({ name: "Test Author" });

      expect(api.createAuthor).toHaveBeenCalledWith({ name: "Test Author" });
      expect(result).toEqual(fakeAuthor);
      expect(authorStore.authors).toEqual([fakeAuthor]);
    });

    it("appends to existing authors", async () => {
      const fakeAuthor2: Author = { ...fakeAuthor, id: "a2", name: "Second" };
      authorStore.authors = [fakeAuthor];
      vi.mocked(api.createAuthor).mockResolvedValue(fakeAuthor2);

      await authorStore.add({ name: "Second" });

      expect(authorStore.authors).toEqual([fakeAuthor, fakeAuthor2]);
    });

    it("propagates errors and leaves authors unchanged", async () => {
      authorStore.authors = [fakeAuthor];
      vi.mocked(api.createAuthor).mockRejectedValue(new Error("conflict"));

      await expect(authorStore.add({ name: "Bad" })).rejects.toThrow(
        "conflict",
      );
      expect(authorStore.authors).toEqual([fakeAuthor]);
    });
  });

  describe("edit", () => {
    it("replaces the updated author in the store", async () => {
      const updated: Author = { ...fakeAuthor, name: "Updated Author" };
      const fakeAuthor2: Author = { ...fakeAuthor, id: "a2", name: "Second" };
      authorStore.authors = [fakeAuthor, fakeAuthor2];
      vi.mocked(api.updateAuthor).mockResolvedValue(updated);

      const result = await authorStore.edit("a1", { name: "Updated Author" });

      expect(api.updateAuthor).toHaveBeenCalledWith("a1", {
        name: "Updated Author",
      });
      expect(result).toEqual(updated);
      expect(authorStore.authors).toEqual([updated, fakeAuthor2]);
    });

    it("propagates errors and leaves authors unchanged", async () => {
      authorStore.authors = [fakeAuthor];
      vi.mocked(api.updateAuthor).mockRejectedValue(new Error("not found"));

      await expect(authorStore.edit("a1", { name: "Bad" })).rejects.toThrow(
        "not found",
      );
      expect(authorStore.authors).toEqual([fakeAuthor]);
    });
  });

  describe("remove", () => {
    it("removes the author from the store", async () => {
      const fakeAuthor2: Author = { ...fakeAuthor, id: "a2", name: "Second" };
      authorStore.authors = [fakeAuthor, fakeAuthor2];
      vi.mocked(api.deleteAuthor).mockResolvedValue(undefined);

      await authorStore.remove("a1");

      expect(api.deleteAuthor).toHaveBeenCalledWith("a1");
      expect(authorStore.authors).toEqual([fakeAuthor2]);
    });

    it("propagates errors and leaves authors unchanged", async () => {
      authorStore.authors = [fakeAuthor];
      vi.mocked(api.deleteAuthor).mockRejectedValue(new Error("server error"));

      await expect(authorStore.remove("a1")).rejects.toThrow("server error");
      expect(authorStore.authors).toEqual([fakeAuthor]);
    });
  });
});
