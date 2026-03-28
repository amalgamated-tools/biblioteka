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
});
