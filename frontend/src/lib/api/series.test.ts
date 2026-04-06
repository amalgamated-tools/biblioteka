import { describe, it, expect, beforeEach, afterEach, vi, type Mock } from "vitest";
import {
  listSeries,
  getSeries,
  createSeries,
  updateSeries,
  deleteSeries,
  listSeriesBooks,
} from "../api";
import type { Series, PaginatedBooks } from "../../types";

let fetchMock: Mock;

function mockFetchResponse(body: unknown, status = 200) {
  const response = {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? "OK" : "Error",
    headers: new Headers({ "content-type": "application/json" }),
    json: vi.fn().mockResolvedValue(body),
    text: vi.fn().mockResolvedValue(JSON.stringify(body)),
  } as unknown as Response;
  fetchMock.mockResolvedValue(response);
}

const fakeSeries: Series = {
  id: "s1",
  name: "Test Series",
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

beforeEach(() => {
  localStorage.clear();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Series API", () => {
  describe("listSeries", () => {
    it("sends GET /api/series and returns the list", async () => {
      mockFetchResponse([fakeSeries]);

      const result = await listSeries();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/series");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeSeries]);
    });
  });

  describe("getSeries", () => {
    it("sends GET /api/series/:id and returns the series", async () => {
      mockFetchResponse(fakeSeries);

      const result = await getSeries("s1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/series/s1");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeSeries);
    });
  });

  describe("createSeries", () => {
    it("sends POST /api/series with input body and returns the created series", async () => {
      mockFetchResponse(fakeSeries);

      const result = await createSeries({ name: "Test Series" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/series");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ name: "Test Series" });
      expect(result).toEqual(fakeSeries);
    });
  });

  describe("updateSeries", () => {
    it("sends PUT /api/series/:id with input body and returns the updated series", async () => {
      const updated: Series = { ...fakeSeries, name: "Updated" };
      mockFetchResponse(updated);

      const result = await updateSeries("s1", { name: "Updated" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/series/s1");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({ name: "Updated" });
      expect(result).toEqual(updated);
    });
  });

  describe("deleteSeries", () => {
    it("sends DELETE /api/series/:id", async () => {
      const resp = {
        ok: true,
        status: 204,
        statusText: "No Content",
        headers: new Headers(),
        json: vi.fn(),
        text: vi.fn(),
      } as unknown as Response;
      fetchMock.mockResolvedValue(resp);

      const result = await deleteSeries("s1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/series/s1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });

  describe("listSeriesBooks", () => {
    it("sends GET with default limit and offset", async () => {
      const paginatedBooks: PaginatedBooks = {
        books: [],
        total: 0,
        limit: 50,
        offset: 0,
      };
      mockFetchResponse(paginatedBooks);

      const result = await listSeriesBooks("s1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/series/s1/books?limit=50&offset=0");
      expect(options.method).toBe("GET");
      expect(result).toEqual(paginatedBooks);
    });

    it("passes custom limit and offset", async () => {
      mockFetchResponse({ books: [], total: 10, limit: 10, offset: 20 });

      await listSeriesBooks("s1", 10, 20);

      const [url] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/series/s1/books?limit=10&offset=20");
    });
  });
});
