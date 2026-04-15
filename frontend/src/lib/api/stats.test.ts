import {
  describe,
  it,
  expect,
  beforeEach,
  afterEach,
  vi,
  type Mock,
} from "vitest";
import { getDownloadsPerMonth, getYearInBooks, clearToken } from "../api";
import type { MonthlyDownloads, YearInBooks } from "../../types";
import { mockFetchResponse } from "./testUtils";

let fetchMock: Mock;

function mockFetch(body: unknown, status = 200) {
  mockFetchResponse(fetchMock, body, status);
}

const fakeDownloads: MonthlyDownloads[] = [
  { month: "2026-01", count: 3 },
  { month: "2026-02", count: 7 },
  { month: "2026-03", count: 0 },
];

const fakeYearInBooks: YearInBooks = {
  year: 2026,
  books_finished: 5,
  active_days: 42,
  longest_streak: 7,
  total_downloads: 12,
};

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Stats API", () => {
  describe("getDownloadsPerMonth", () => {
    it("sends GET /api/stats/downloads-per-month with default months", async () => {
      mockFetch(fakeDownloads);

      const result = await getDownloadsPerMonth();

      expect(fetchMock).toHaveBeenCalledOnce();
      const url: string = fetchMock.mock.calls[0][0] as string;
      expect(url).toContain("/api/stats/downloads-per-month");
      expect(url).toContain("months=12");
      expect(result).toEqual(fakeDownloads);
    });

    it("sends the requested number of months as a query parameter", async () => {
      mockFetch(fakeDownloads);

      await getDownloadsPerMonth(6);

      const url: string = fetchMock.mock.calls[0][0] as string;
      expect(url).toContain("months=6");
    });

    it("returns the response array directly", async () => {
      mockFetch(fakeDownloads);

      const result = await getDownloadsPerMonth(3);
      expect(result).toHaveLength(3);
      expect(result[0]).toEqual({ month: "2026-01", count: 3 });
    });
  });

  describe("getYearInBooks", () => {
    it("sends GET /api/stats/year-in-books without a year param by default", async () => {
      mockFetch(fakeYearInBooks);

      const result = await getYearInBooks();

      expect(fetchMock).toHaveBeenCalledOnce();
      const url: string = fetchMock.mock.calls[0][0] as string;
      expect(url).toContain("/api/stats/year-in-books");
      expect(url).not.toContain("year=");
      expect(result).toEqual(fakeYearInBooks);
    });

    it("includes year query param when provided", async () => {
      mockFetch(fakeYearInBooks);

      await getYearInBooks(2025);

      const url: string = fetchMock.mock.calls[0][0] as string;
      expect(url).toContain("year=2025");
    });

    it("returns the response object directly", async () => {
      mockFetch(fakeYearInBooks);

      const result = await getYearInBooks(2026);
      expect(result.year).toBe(2026);
      expect(result.books_finished).toBe(5);
      expect(result.active_days).toBe(42);
      expect(result.longest_streak).toBe(7);
      expect(result.total_downloads).toBe(12);
    });
  });
});
