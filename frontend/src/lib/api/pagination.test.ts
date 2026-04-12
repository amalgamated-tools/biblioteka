import {
  describe,
  it,
  expect,
  beforeEach,
  afterEach,
  vi,
  type Mock,
} from "vitest";
import { listEntityBooks, clearToken } from "../api";
import type { PaginatedBooks } from "../../types";
import { mockFetchResponse as _mockFetchResponse } from "./testUtils";

let fetchMock: Mock;

function mockFetchResponse(body: unknown, status = 200) {
  _mockFetchResponse(fetchMock, body, status);
}

const fakePaginatedBooks: PaginatedBooks = {
  books: [],
  total: 0,
  limit: 50,
  offset: 0,
};

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Pagination API", () => {
  describe("listEntityBooks", () => {
    it("constructs the correct URL with the /books suffix and limit/offset query parameters", async () => {
      mockFetchResponse(fakePaginatedBooks);

      const result = await listEntityBooks("/api/authors/a1", 50, 0);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/authors/a1/books?limit=50&offset=0");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakePaginatedBooks);
    });

    it("passes custom limit and offset in the query string", async () => {
      mockFetchResponse({ books: [], total: 10, limit: 10, offset: 20 });

      await listEntityBooks("/api/series/s1", 10, 20);

      const [url] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/series/s1/books?limit=10&offset=20");
    });
  });
});
