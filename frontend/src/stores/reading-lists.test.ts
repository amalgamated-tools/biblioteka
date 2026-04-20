import { describe, it, expect, beforeEach, vi } from "vitest";
import { readingListStore } from "./reading-lists.svelte";
import * as api from "../lib/api";
import type { ReadingList } from "../types";

vi.mock("../lib/api", async () => {
  return {
    listReadingLists: vi.fn(),
    createReadingList: vi.fn(),
    updateReadingList: vi.fn(),
    deleteReadingList: vi.fn(),
    addBookToReadingList: vi.fn(),
    removeBookFromReadingList: vi.fn(),
  };
});

const fakeList: ReadingList = {
  id: "rl-1",
  name: "To Read",
  description: null,
  book_count: 0,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const fakeList2: ReadingList = {
  id: "rl-2",
  name: "Favorites",
  description: "My favorites",
  book_count: 5,
  created_at: "2026-02-01T00:00:00Z",
  updated_at: "2026-02-01T00:00:00Z",
};

describe("reading list store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    readingListStore.lists = [];
    readingListStore.loading = false;
    readingListStore.loaded = false;
    readingListStore.loadError = null;
  });

  describe("load", () => {
    it("fetches reading lists and sets loaded", async () => {
      vi.mocked(api.listReadingLists).mockResolvedValue([fakeList]);

      await readingListStore.load();

      expect(api.listReadingLists).toHaveBeenCalledTimes(1);
      expect(readingListStore.lists).toEqual([fakeList]);
      expect(readingListStore.loaded).toBe(true);
      expect(readingListStore.loading).toBe(false);
    });

    it("does not call API again when already loaded", async () => {
      vi.mocked(api.listReadingLists).mockResolvedValue([fakeList]);

      await readingListStore.load();
      await readingListStore.load();

      expect(api.listReadingLists).toHaveBeenCalledTimes(1);
    });

    it("does not call API concurrently when loading is in progress", async () => {
      vi.mocked(api.listReadingLists).mockResolvedValue([fakeList]);

      await Promise.all([
        readingListStore.load(),
        readingListStore.load(),
        readingListStore.load(),
      ]);

      expect(api.listReadingLists).toHaveBeenCalledTimes(1);
    });

    it("sets loadError and marks loaded on API error", async () => {
      vi.mocked(api.listReadingLists).mockRejectedValue(
        new Error("network error"),
      );

      await readingListStore.load();

      expect(readingListStore.loading).toBe(false);
      expect(readingListStore.loaded).toBe(true);
      expect(readingListStore.loadError).toBe("network error");
      expect(readingListStore.lists).toEqual([]);
    });

    it("uses fallback error message for non-Error rejections", async () => {
      vi.mocked(api.listReadingLists).mockRejectedValue("oops");

      await readingListStore.load();

      expect(readingListStore.loadError).toBe("failed to load reading lists");
    });
  });

  describe("reload", () => {
    it("forces a reload even when already loaded", async () => {
      vi.mocked(api.listReadingLists).mockResolvedValue([fakeList]);

      await readingListStore.load();
      await readingListStore.reload();

      expect(api.listReadingLists).toHaveBeenCalledTimes(2);
    });
  });

  describe("create", () => {
    it("appends the created list sorted by name", async () => {
      readingListStore.lists = [fakeList2];
      vi.mocked(api.createReadingList).mockResolvedValue(fakeList);

      const result = await readingListStore.create({ name: "To Read" });

      expect(api.createReadingList).toHaveBeenCalledWith({ name: "To Read" });
      expect(result).toEqual(fakeList);
      // "Favorites" comes before "To Read" alphabetically
      expect(readingListStore.lists).toEqual([fakeList2, fakeList]);
    });

    it("creates first list when store is empty", async () => {
      vi.mocked(api.createReadingList).mockResolvedValue(fakeList);

      const result = await readingListStore.create({ name: "To Read" });

      expect(result).toEqual(fakeList);
      expect(readingListStore.lists).toEqual([fakeList]);
    });

    it("propagates errors and leaves lists unchanged", async () => {
      readingListStore.lists = [fakeList];
      vi.mocked(api.createReadingList).mockRejectedValue(new Error("conflict"));

      await expect(readingListStore.create({ name: "Bad" })).rejects.toThrow(
        "conflict",
      );
      expect(readingListStore.lists).toEqual([fakeList]);
    });

    it("creates with description when provided", async () => {
      vi.mocked(api.createReadingList).mockResolvedValue(fakeList2);

      await readingListStore.create({
        name: "Favorites",
        description: "My favorites",
      });

      expect(api.createReadingList).toHaveBeenCalledWith({
        name: "Favorites",
        description: "My favorites",
      });
    });
  });

  describe("update", () => {
    it("replaces the updated list and re-sorts by name", async () => {
      const updatedList: ReadingList = { ...fakeList, name: "Already Read" };
      readingListStore.lists = [fakeList, fakeList2];
      vi.mocked(api.updateReadingList).mockResolvedValue(updatedList);

      const result = await readingListStore.update("rl-1", {
        name: "Already Read",
      });

      expect(api.updateReadingList).toHaveBeenCalledWith("rl-1", {
        name: "Already Read",
      });
      expect(result).toEqual(updatedList);
      // "Already Read" comes before "Favorites" alphabetically
      expect(readingListStore.lists).toEqual([updatedList, fakeList2]);
    });

    it("propagates errors and leaves lists unchanged", async () => {
      readingListStore.lists = [fakeList];
      vi.mocked(api.updateReadingList).mockRejectedValue(
        new Error("not found"),
      );

      await expect(
        readingListStore.update("rl-1", { name: "Bad" }),
      ).rejects.toThrow("not found");
      expect(readingListStore.lists).toEqual([fakeList]);
    });
  });

  describe("remove", () => {
    it("removes the list from the store after deletion", async () => {
      readingListStore.lists = [fakeList, fakeList2];
      vi.mocked(api.deleteReadingList).mockResolvedValue(undefined);

      await readingListStore.remove("rl-1");

      expect(api.deleteReadingList).toHaveBeenCalledWith("rl-1");
      expect(readingListStore.lists).toEqual([fakeList2]);
    });

    it("propagates errors and leaves lists unchanged", async () => {
      readingListStore.lists = [fakeList];
      vi.mocked(api.deleteReadingList).mockRejectedValue(
        new Error("server error"),
      );

      await expect(readingListStore.remove("rl-1")).rejects.toThrow(
        "server error",
      );
      expect(readingListStore.lists).toEqual([fakeList]);
    });
  });

  describe("addBook", () => {
    it("calls addBookToReadingList and reloads the store", async () => {
      readingListStore.lists = [fakeList];
      readingListStore.loaded = true;

      const updatedList: ReadingList = { ...fakeList, book_count: 1 };
      vi.mocked(api.addBookToReadingList).mockResolvedValue(undefined);
      vi.mocked(api.listReadingLists).mockResolvedValue([updatedList]);

      await readingListStore.addBook("rl-1", "b-1");

      expect(api.addBookToReadingList).toHaveBeenCalledWith("rl-1", "b-1");
      expect(api.listReadingLists).toHaveBeenCalledTimes(1);
      expect(readingListStore.lists).toEqual([updatedList]);
    });

    it("propagates errors from addBookToReadingList", async () => {
      vi.mocked(api.addBookToReadingList).mockRejectedValue(
        new Error("not found"),
      );

      await expect(readingListStore.addBook("rl-1", "b-1")).rejects.toThrow(
        "not found",
      );
    });
  });

  describe("removeBook", () => {
    it("calls removeBookFromReadingList and reloads the store", async () => {
      readingListStore.lists = [{ ...fakeList, book_count: 1 }];
      readingListStore.loaded = true;

      vi.mocked(api.removeBookFromReadingList).mockResolvedValue(undefined);
      vi.mocked(api.listReadingLists).mockResolvedValue([fakeList]);

      await readingListStore.removeBook("rl-1", "b-1");

      expect(api.removeBookFromReadingList).toHaveBeenCalledWith("rl-1", "b-1");
      expect(api.listReadingLists).toHaveBeenCalledTimes(1);
      expect(readingListStore.lists).toEqual([fakeList]);
    });

    it("propagates errors from removeBookFromReadingList", async () => {
      vi.mocked(api.removeBookFromReadingList).mockRejectedValue(
        new Error("not found"),
      );

      await expect(readingListStore.removeBook("rl-1", "b-1")).rejects.toThrow(
        "not found",
      );
    });
  });
});
