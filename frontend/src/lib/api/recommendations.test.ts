import {
  describe,
  it,
  expect,
  beforeEach,
  afterEach,
  vi,
  type Mock,
} from "vitest";
import { getRecommendations, clearToken } from "../api";
import type { BookSummary } from "../../types";
import { mockFetchResponse } from "./testUtils";

let fetchMock: Mock;

function mockFetch(body: unknown, status = 200) {
  mockFetchResponse(fetchMock, body, status);
}

const fakeBook: BookSummary = {
  id: "book1",
  title: "Foundation",
  description: null,
  asin: null,
  isbn10: null,
  isbn13: null,
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  publication_date: null,
  publisher: null,
  language: null,
  cover_image_url: null,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Recommendations API", () => {
  describe("getRecommendations", () => {
    it("sends GET /api/recommendations with default limit of 10", async () => {
      mockFetch([fakeBook]);

      const result = await getRecommendations();

      expect(fetchMock).toHaveBeenCalledOnce();
      const url: string = fetchMock.mock.calls[0][0] as string;
      expect(url).toContain("/api/recommendations");
      expect(url).toContain("limit=10");
      expect(fetchMock.mock.calls[0][1].method).toBe("GET");
      expect(result).toEqual([fakeBook]);
    });

    it("sends the requested limit as a query parameter", async () => {
      mockFetch([fakeBook]);

      await getRecommendations(25);

      const url: string = fetchMock.mock.calls[0][0] as string;
      expect(url).toContain("limit=25");
    });

    it("returns an empty array when no recommendations are available", async () => {
      mockFetch([]);

      const result = await getRecommendations();

      expect(result).toEqual([]);
    });

    it("returns multiple recommendations", async () => {
      const books = [fakeBook, { ...fakeBook, id: "book2", title: "Dune" }];
      mockFetch(books);

      const result = await getRecommendations(2);

      expect(result).toHaveLength(2);
      expect(result[0].title).toBe("Foundation");
      expect(result[1].title).toBe("Dune");
    });

    it("passes an AbortSignal to fetch when provided", async () => {
      mockFetch([]);
      const controller = new AbortController();

      await getRecommendations(10, controller.signal);

      const options = fetchMock.mock.calls[0][1] as RequestInit;
      expect(options.signal).toBe(controller.signal);
    });
  });
});
