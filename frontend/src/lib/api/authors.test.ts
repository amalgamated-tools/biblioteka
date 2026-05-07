import {
  describe,
  it,
  expect,
  beforeEach,
  afterEach,
  vi,
  type Mock,
} from "vitest";
import {
  listAuthors,
  getAuthor,
  createAuthor,
  updateAuthor,
  deleteAuthor,
  listAuthorBooks,
  clearToken,
} from "../api";
import type { Author, PaginatedBooks } from "../../types";
import {
  mockFetchResponse as _mockFetchResponse,
  mockNoContentResponse as _mockNoContentResponse,
} from "./testUtils";

let fetchMock: Mock;

function mockFetchResponse(body: unknown, status = 200) {
  _mockFetchResponse(fetchMock, body, status);
}

function mockNoContentResponse() {
  _mockNoContentResponse(fetchMock);
}

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

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Authors API", () => {
  describe("listAuthors", () => {
    it("fetches all pages and returns the full list", async () => {
      const fakeAuthor2: Author = {
        ...fakeAuthor,
        id: "a2",
        name: "Second Author",
      };
      const makeResponse = (body: unknown): Response =>
        ({
          ok: true,
          status: 200,
          statusText: "OK",
          headers: new Headers({ "content-type": "application/json" }),
          json: vi.fn().mockResolvedValue(body),
          text: vi.fn().mockResolvedValue(JSON.stringify(body)),
        }) as unknown as Response;
      fetchMock.mockResolvedValueOnce(
        makeResponse({
          authors: [fakeAuthor],
          total: 2,
          limit: 200,
          offset: 0,
        }),
      );
      fetchMock.mockResolvedValueOnce(
        makeResponse({
          authors: [fakeAuthor2],
          total: 2,
          limit: 200,
          offset: 1,
        }),
      );

      const result = await listAuthors();

      const [firstURL, firstOptions] = fetchMock.mock.calls[0];
      const [secondURL, secondOptions] = fetchMock.mock.calls[1];
      expect(firstURL).toBe("/api/authors?limit=200&offset=0");
      expect(firstOptions.method).toBe("GET");
      expect(secondURL).toBe("/api/authors?limit=200&offset=1");
      expect(secondOptions.method).toBe("GET");
      expect(result).toEqual([fakeAuthor, fakeAuthor2]);
    });
  });

  describe("getAuthor", () => {
    it("sends GET /api/authors/:id and returns the author", async () => {
      mockFetchResponse(fakeAuthor);

      const result = await getAuthor("a1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/authors/a1");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeAuthor);
    });
  });

  describe("createAuthor", () => {
    it("sends POST /api/authors with input body and returns the created author", async () => {
      mockFetchResponse(fakeAuthor);

      const result = await createAuthor({ name: "Test Author" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/authors");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ name: "Test Author" });
      expect(result).toEqual(fakeAuthor);
    });
  });

  describe("updateAuthor", () => {
    it("sends PUT /api/authors/:id with input body and returns the updated author", async () => {
      const updated: Author = { ...fakeAuthor, name: "Updated" };
      mockFetchResponse(updated);

      const result = await updateAuthor("a1", { name: "Updated" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/authors/a1");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({ name: "Updated" });
      expect(result).toEqual(updated);
    });
  });

  describe("deleteAuthor", () => {
    it("sends DELETE /api/authors/:id", async () => {
      mockNoContentResponse();

      const result = await deleteAuthor("a1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/authors/a1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });

  describe("listAuthorBooks", () => {
    it("sends GET with default limit and offset", async () => {
      const paginatedBooks: PaginatedBooks = {
        books: [],
        total: 0,
        limit: 50,
        offset: 0,
      };
      mockFetchResponse(paginatedBooks);

      const result = await listAuthorBooks("a1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/authors/a1/books?limit=50&offset=0");
      expect(options.method).toBe("GET");
      expect(result).toEqual(paginatedBooks);
    });

    it("passes custom limit and offset", async () => {
      mockFetchResponse({ books: [], total: 10, limit: 10, offset: 20 });

      await listAuthorBooks("a1", 10, 20);

      const [url] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/authors/a1/books?limit=10&offset=20");
    });
  });
});
