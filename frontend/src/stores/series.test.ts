import { describe, it, expect, beforeEach, vi } from "vitest";
import { seriesStore } from "./series.svelte";
import * as api from "../lib/api";
import type { Series } from "../types";

vi.mock("../lib/api", async () => {
  return {
    listSeries: vi.fn(),
    createSeries: vi.fn(),
    updateSeries: vi.fn(),
    deleteSeries: vi.fn(),
  };
});

const fakeSeries: Series = {
  id: "s1",
  name: "Test Series",
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("series store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    seriesStore.series = [];
    seriesStore.loading = false;
    seriesStore.loaded = false;
  });

  describe("load", () => {
    it("fetches series and sets loaded", async () => {
      vi.mocked(api.listSeries).mockResolvedValue([fakeSeries]);

      await seriesStore.load();

      expect(api.listSeries).toHaveBeenCalledTimes(1);
      expect(seriesStore.series).toEqual([fakeSeries]);
      expect(seriesStore.loaded).toBe(true);
      expect(seriesStore.loading).toBe(false);
    });

    it("does not call API again after already loaded", async () => {
      vi.mocked(api.listSeries).mockResolvedValue([fakeSeries]);

      await seriesStore.load();
      await seriesStore.load();

      expect(api.listSeries).toHaveBeenCalledTimes(1);
    });

    it("calls API only once when invoked concurrently", async () => {
      vi.mocked(api.listSeries).mockResolvedValue([fakeSeries]);

      await Promise.all([
        seriesStore.load(),
        seriesStore.load(),
        seriesStore.load(),
      ]);

      expect(api.listSeries).toHaveBeenCalledTimes(1);
      expect(seriesStore.series).toEqual([fakeSeries]);
      expect(seriesStore.loaded).toBe(true);
    });

    it("resets loading on API error", async () => {
      vi.mocked(api.listSeries).mockRejectedValue(new Error("fail"));

      await seriesStore.load();

      expect(seriesStore.loading).toBe(false);
      expect(seriesStore.loaded).toBe(false);
      expect(seriesStore.series).toEqual([]);
    });
  });

  describe("add", () => {
    it("appends the created series to the store", async () => {
      vi.mocked(api.createSeries).mockResolvedValue(fakeSeries);

      const result = await seriesStore.add({ name: "Test Series" });

      expect(api.createSeries).toHaveBeenCalledWith({ name: "Test Series" });
      expect(result).toEqual(fakeSeries);
      expect(seriesStore.series).toEqual([fakeSeries]);
    });

    it("appends to existing series", async () => {
      const fakeSeries2: Series = { ...fakeSeries, id: "s2", name: "Second" };
      seriesStore.series = [fakeSeries];
      vi.mocked(api.createSeries).mockResolvedValue(fakeSeries2);

      await seriesStore.add({ name: "Second" });

      expect(seriesStore.series).toEqual([fakeSeries, fakeSeries2]);
    });

    it("propagates errors and leaves series unchanged", async () => {
      seriesStore.series = [fakeSeries];
      vi.mocked(api.createSeries).mockRejectedValue(new Error("conflict"));

      await expect(seriesStore.add({ name: "Bad" })).rejects.toThrow(
        "conflict",
      );
      expect(seriesStore.series).toEqual([fakeSeries]);
    });
  });

  describe("edit", () => {
    it("replaces the updated series in the store", async () => {
      const updated: Series = { ...fakeSeries, name: "Updated Series" };
      const fakeSeries2: Series = { ...fakeSeries, id: "s2", name: "Second" };
      seriesStore.series = [fakeSeries, fakeSeries2];
      vi.mocked(api.updateSeries).mockResolvedValue(updated);

      const result = await seriesStore.edit("s1", { name: "Updated Series" });

      expect(api.updateSeries).toHaveBeenCalledWith("s1", {
        name: "Updated Series",
      });
      expect(result).toEqual(updated);
      expect(seriesStore.series).toEqual([updated, fakeSeries2]);
    });

    it("propagates errors and leaves series unchanged", async () => {
      seriesStore.series = [fakeSeries];
      vi.mocked(api.updateSeries).mockRejectedValue(new Error("not found"));

      await expect(seriesStore.edit("s1", { name: "Bad" })).rejects.toThrow(
        "not found",
      );
      expect(seriesStore.series).toEqual([fakeSeries]);
    });
  });

  describe("remove", () => {
    it("removes the series from the store", async () => {
      const fakeSeries2: Series = { ...fakeSeries, id: "s2", name: "Second" };
      seriesStore.series = [fakeSeries, fakeSeries2];
      vi.mocked(api.deleteSeries).mockResolvedValue(undefined);

      await seriesStore.remove("s1");

      expect(api.deleteSeries).toHaveBeenCalledWith("s1");
      expect(seriesStore.series).toEqual([fakeSeries2]);
    });

    it("propagates errors and leaves series unchanged", async () => {
      seriesStore.series = [fakeSeries];
      vi.mocked(api.deleteSeries).mockRejectedValue(new Error("server error"));

      await expect(seriesStore.remove("s1")).rejects.toThrow("server error");
      expect(seriesStore.series).toEqual([fakeSeries]);
    });
  });
});
