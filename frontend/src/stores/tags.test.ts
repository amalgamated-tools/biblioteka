import { describe, it, expect, beforeEach, vi } from "vitest";
import { tagStore } from "./tags.svelte";
import * as api from "../lib/api";
import type { Tag } from "../types";

vi.mock("../lib/api", async () => {
  return {
    listTags: vi.fn(),
    createTag: vi.fn(),
    updateTag: vi.fn(),
    deleteTag: vi.fn(),
  };
});

const fakeTag: Tag = {
  id: "t1",
  name: "fiction",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("tag store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    tagStore.tags = [];
    tagStore.loading = false;
    tagStore.loaded = false;
  });

  describe("load", () => {
    it("fetches tags and sets loaded", async () => {
      vi.mocked(api.listTags).mockResolvedValue([fakeTag]);

      await tagStore.load();

      expect(api.listTags).toHaveBeenCalledTimes(1);
      expect(tagStore.tags).toEqual([fakeTag]);
      expect(tagStore.loaded).toBe(true);
      expect(tagStore.loading).toBe(false);
    });

    it("does not call API again after already loaded", async () => {
      vi.mocked(api.listTags).mockResolvedValue([fakeTag]);

      await tagStore.load();
      await tagStore.load();

      expect(api.listTags).toHaveBeenCalledTimes(1);
    });

    it("calls API only once when invoked concurrently", async () => {
      vi.mocked(api.listTags).mockResolvedValue([fakeTag]);

      await Promise.all([tagStore.load(), tagStore.load(), tagStore.load()]);

      expect(api.listTags).toHaveBeenCalledTimes(1);
      expect(tagStore.tags).toEqual([fakeTag]);
      expect(tagStore.loaded).toBe(true);
    });

    it("resets loading on API error", async () => {
      vi.mocked(api.listTags).mockRejectedValue(new Error("fail"));

      await tagStore.load();

      expect(tagStore.loading).toBe(false);
      expect(tagStore.loaded).toBe(false);
      expect(tagStore.tags).toEqual([]);
    });
  });

  describe("add", () => {
    it("appends the created tag to the store", async () => {
      vi.mocked(api.createTag).mockResolvedValue(fakeTag);

      const result = await tagStore.add({ name: "fiction" });

      expect(api.createTag).toHaveBeenCalledWith({ name: "fiction" });
      expect(result).toEqual(fakeTag);
      expect(tagStore.tags).toEqual([fakeTag]);
    });

    it("appends to existing tags", async () => {
      const fakeTag2: Tag = { ...fakeTag, id: "t2", name: "fantasy" };
      tagStore.tags = [fakeTag];
      vi.mocked(api.createTag).mockResolvedValue(fakeTag2);

      await tagStore.add({ name: "fantasy" });

      expect(tagStore.tags).toEqual([fakeTag, fakeTag2]);
    });

    it("propagates errors and leaves tags unchanged", async () => {
      tagStore.tags = [fakeTag];
      vi.mocked(api.createTag).mockRejectedValue(new Error("conflict"));

      await expect(tagStore.add({ name: "bad" })).rejects.toThrow("conflict");
      expect(tagStore.tags).toEqual([fakeTag]);
    });
  });

  describe("edit", () => {
    it("replaces the updated tag in the store", async () => {
      const updated: Tag = { ...fakeTag, name: "literary fiction" };
      const fakeTag2: Tag = { ...fakeTag, id: "t2", name: "fantasy" };
      tagStore.tags = [fakeTag, fakeTag2];
      vi.mocked(api.updateTag).mockResolvedValue(updated);

      const result = await tagStore.edit("t1", { name: "literary fiction" });

      expect(api.updateTag).toHaveBeenCalledWith("t1", {
        name: "literary fiction",
      });
      expect(result).toEqual(updated);
      expect(tagStore.tags).toEqual([updated, fakeTag2]);
    });

    it("propagates errors and leaves tags unchanged", async () => {
      tagStore.tags = [fakeTag];
      vi.mocked(api.updateTag).mockRejectedValue(new Error("not found"));

      await expect(tagStore.edit("t1", { name: "bad" })).rejects.toThrow(
        "not found",
      );
      expect(tagStore.tags).toEqual([fakeTag]);
    });
  });

  describe("remove", () => {
    it("removes the tag from the store", async () => {
      const fakeTag2: Tag = { ...fakeTag, id: "t2", name: "fantasy" };
      tagStore.tags = [fakeTag, fakeTag2];
      vi.mocked(api.deleteTag).mockResolvedValue(undefined);

      await tagStore.remove("t1");

      expect(api.deleteTag).toHaveBeenCalledWith("t1");
      expect(tagStore.tags).toEqual([fakeTag2]);
    });

    it("propagates errors and leaves tags unchanged", async () => {
      tagStore.tags = [fakeTag];
      vi.mocked(api.deleteTag).mockRejectedValue(new Error("server error"));

      await expect(tagStore.remove("t1")).rejects.toThrow("server error");
      expect(tagStore.tags).toEqual([fakeTag]);
    });
  });
});
