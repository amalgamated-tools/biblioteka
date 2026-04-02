import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { libraryStore } from "./libraries.svelte";
import * as api from "../lib/api";
import type { Library } from "../types";

vi.mock("../lib/api", async () => {
  return {
    listLibraries: vi.fn(),
    createLibrary: vi.fn(),
    updateLibrary: vi.fn(),
    deleteLibrary: vi.fn(),
  };
});

const fakeLibrary: Library = {
  id: "lib1",
  name: "Test Library",
  paths: ["/books"],
  organization_type: "book_per_folder",
  monitored: false,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("library store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    libraryStore.libraries = [];
    libraryStore.loading = false;
    libraryStore.loaded = false;
    libraryStore.scanningIds.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("load", () => {
    it("fetches libraries and sets loaded", async () => {
      vi.mocked(api.listLibraries).mockResolvedValue([fakeLibrary]);

      await libraryStore.load();

      expect(api.listLibraries).toHaveBeenCalledTimes(1);
      expect(libraryStore.libraries).toEqual([fakeLibrary]);
      expect(libraryStore.loaded).toBe(true);
      expect(libraryStore.loading).toBe(false);
    });

    it("does not call API again after already loaded", async () => {
      vi.mocked(api.listLibraries).mockResolvedValue([fakeLibrary]);

      await libraryStore.load();
      await libraryStore.load();

      expect(api.listLibraries).toHaveBeenCalledTimes(1);
    });

    it("resets loading on API error", async () => {
      vi.mocked(api.listLibraries).mockRejectedValue(new Error("fail"));

      await libraryStore.load();

      expect(libraryStore.loading).toBe(false);
      expect(libraryStore.loaded).toBe(false);
      expect(libraryStore.libraries).toEqual([]);
    });
  });

  describe("add", () => {
    it("adds the created library to the store", async () => {
      vi.mocked(api.createLibrary).mockResolvedValue(fakeLibrary);

      const result = await libraryStore.add({
        name: "Test Library",
        paths: ["/books"],
        organization_type: "book_per_folder",
        monitored: false,
      });

      expect(result).toEqual(fakeLibrary);
      expect(libraryStore.libraries).toEqual([fakeLibrary]);
    });

    it("marks the newly added library as scanning", async () => {
      vi.mocked(api.createLibrary).mockResolvedValue(fakeLibrary);

      await libraryStore.add({
        name: "Test Library",
        paths: ["/books"],
        organization_type: "book_per_folder",
        monitored: false,
      });

      expect(libraryStore.scanningIds.has("lib1")).toBe(true);
      expect(libraryStore.isScanning).toBe(true);
    });

    it("auto-clears scanning state after timeout", async () => {
      vi.mocked(api.createLibrary).mockResolvedValue(fakeLibrary);

      await libraryStore.add({
        name: "Test Library",
        paths: ["/books"],
        organization_type: "book_per_folder",
        monitored: false,
      });

      expect(libraryStore.scanningIds.has("lib1")).toBe(true);

      // Fast-forward past the 5-minute timeout
      vi.advanceTimersByTime(5 * 60 * 1000 + 1);

      expect(libraryStore.scanningIds.has("lib1")).toBe(false);
      expect(libraryStore.isScanning).toBe(false);
    });
  });

  describe("clearScanning", () => {
    it("removes the specified library ID from scanningIds", async () => {
      vi.mocked(api.createLibrary).mockResolvedValue(fakeLibrary);
      await libraryStore.add({
        name: "Test Library",
        paths: ["/books"],
        organization_type: "book_per_folder",
        monitored: false,
      });
      expect(libraryStore.scanningIds.has("lib1")).toBe(true);

      libraryStore.clearScanning("lib1");

      expect(libraryStore.scanningIds.has("lib1")).toBe(false);
      expect(libraryStore.isScanning).toBe(false);
    });

    it("is a no-op when ID is not in scanningIds", () => {
      libraryStore.clearScanning("nonexistent");
      expect(libraryStore.isScanning).toBe(false);
    });
  });

  describe("clearAllScanning", () => {
    it("clears all scanning IDs", async () => {
      const lib2: Library = { ...fakeLibrary, id: "lib2" };
      vi.mocked(api.createLibrary)
        .mockResolvedValueOnce(fakeLibrary)
        .mockResolvedValueOnce(lib2);

      await libraryStore.add({
        name: "Lib 1",
        paths: ["/books1"],
        organization_type: "book_per_folder",
        monitored: false,
      });
      await libraryStore.add({
        name: "Lib 2",
        paths: ["/books2"],
        organization_type: "book_per_folder",
        monitored: false,
      });
      expect(libraryStore.scanningIds.size).toBe(2);

      libraryStore.clearAllScanning();

      expect(libraryStore.scanningIds.size).toBe(0);
      expect(libraryStore.isScanning).toBe(false);
    });
  });

  describe("remove", () => {
    it("removes the library from the store and clears its scanning state", async () => {
      vi.mocked(api.createLibrary).mockResolvedValue(fakeLibrary);
      vi.mocked(api.deleteLibrary).mockResolvedValue(undefined);

      await libraryStore.add({
        name: "Test Library",
        paths: ["/books"],
        organization_type: "book_per_folder",
        monitored: false,
      });
      expect(libraryStore.scanningIds.has("lib1")).toBe(true);

      await libraryStore.remove("lib1");

      expect(libraryStore.libraries).toEqual([]);
      expect(libraryStore.scanningIds.has("lib1")).toBe(false);
    });
  });
});
