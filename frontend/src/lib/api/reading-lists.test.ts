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
  listReadingLists,
  getReadingList,
  createReadingList,
  updateReadingList,
  deleteReadingList,
  listReadingListBooks,
  addBookToReadingList,
  removeBookFromReadingList,
  getReadingListsForBook,
  clearToken,
} from "../api";
import type { ReadingList, PaginatedBooks } from "../../types";
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

const fakeList: ReadingList = {
  id: "rl-1",
  name: "To Read",
  description: null,
  book_count: 3,
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

describe("Reading Lists API", () => {
  describe("listReadingLists", () => {
    it("sends GET /api/reading-lists and returns the list", async () => {
      mockFetchResponse([fakeList]);

      const result = await listReadingLists();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/reading-lists");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeList]);
    });
  });

  describe("getReadingList", () => {
    it("sends GET /api/reading-lists/:id and returns the list", async () => {
      mockFetchResponse(fakeList);

      const result = await getReadingList("rl-1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/reading-lists/rl-1");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeList);
    });
  });

  describe("createReadingList", () => {
    it("sends POST /api/reading-lists with body and returns the created list", async () => {
      mockFetchResponse(fakeList, 201);

      const result = await createReadingList({ name: "To Read" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/reading-lists");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ name: "To Read" });
      expect(result).toEqual(fakeList);
    });

    it("passes description when provided", async () => {
      mockFetchResponse(fakeList, 201);

      await createReadingList({ name: "To Read", description: "My list" });

      const [, options] = fetchMock.mock.calls[0];
      expect(JSON.parse(options.body)).toEqual({
        name: "To Read",
        description: "My list",
      });
    });
  });

  describe("updateReadingList", () => {
    it("sends PUT /api/reading-lists/:id with updated body", async () => {
      const updated: ReadingList = { ...fakeList, name: "Updated" };
      mockFetchResponse(updated);

      const result = await updateReadingList("rl-1", { name: "Updated" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/reading-lists/rl-1");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({ name: "Updated" });
      expect(result).toEqual(updated);
    });
  });

  describe("deleteReadingList", () => {
    it("sends DELETE /api/reading-lists/:id", async () => {
      mockNoContentResponse();

      const result = await deleteReadingList("rl-1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/reading-lists/rl-1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });

  describe("listReadingListBooks", () => {
    it("sends GET with default limit and offset", async () => {
      const paginatedBooks: PaginatedBooks = {
        books: [],
        total: 0,
        limit: 50,
        offset: 0,
      };
      mockFetchResponse(paginatedBooks);

      const result = await listReadingListBooks("rl-1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/reading-lists/rl-1/books?limit=50&offset=0");
      expect(options.method).toBe("GET");
      expect(result).toEqual(paginatedBooks);
    });

    it("passes custom limit and offset", async () => {
      mockFetchResponse({ books: [], total: 10, limit: 10, offset: 20 });

      await listReadingListBooks("rl-1", 10, 20);

      const [url] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/reading-lists/rl-1/books?limit=10&offset=20");
    });
  });

  describe("addBookToReadingList", () => {
    it("sends POST /api/reading-lists/:id/books with book_id", async () => {
      mockNoContentResponse();

      await addBookToReadingList("rl-1", "book-42");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/reading-lists/rl-1/books");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ book_id: "book-42" });
    });
  });

  describe("removeBookFromReadingList", () => {
    it("sends DELETE /api/reading-lists/:listId/books/:bookId", async () => {
      mockNoContentResponse();

      await removeBookFromReadingList("rl-1", "book-42");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/reading-lists/rl-1/books/book-42");
      expect(options.method).toBe("DELETE");
    });
  });

  describe("getReadingListsForBook", () => {
    it("sends GET /api/books/:bookId/reading-lists", async () => {
      mockFetchResponse([fakeList]);

      const result = await getReadingListsForBook("book-42");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/book-42/reading-lists");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeList]);
    });
  });
});
