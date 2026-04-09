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
  listBooks,
  getBook,
  createBook,
  updateBook,
  deleteBook,
  getBookAuthors,
  setBookAuthors,
  getBookSeries,
  setBookSeries,
  listBookFiles,
  createBookFile,
  getBookFile,
  deleteBookFile,
  bookFileDownloadUrl,
  emailBookFile,
  clearToken,
} from "../api";
import type {
  Author,
  Book,
  BookFile,
  BookSeriesEntry,
  PaginatedBooks,
  Series,
} from "../../types";
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

const fakeBook: Book = {
  id: "b1",
  title: "Test Book",
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
  authors: [],
  series: [],
  files: [],
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

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

const fakeSeries: Series = {
  id: "s1",
  name: "Test Series",
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const fakeSeriesEntry: BookSeriesEntry = {
  series: fakeSeries,
  position: 1,
};

const fakeBookFile: BookFile = {
  id: "f1",
  book_id: "b1",
  file_type: "epub",
  file_name: "test.epub",
  file_size: 1024,
  file_hash: null,
  file_path: "/books/test.epub",
  download_count: 0,
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

describe("Books API", () => {
  describe("listBooks", () => {
    it("sends GET /api/books with default params", async () => {
      const paginatedBooks: PaginatedBooks = {
        books: [],
        total: 0,
        limit: 50,
        offset: 0,
      };
      mockFetchResponse(paginatedBooks);

      const result = await listBooks();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books?limit=50&offset=0");
      expect(options.method).toBe("GET");
      expect(result).toEqual(paginatedBooks);
    });

    it("passes custom limit, offset, and query", async () => {
      mockFetchResponse({ books: [], total: 0, limit: 10, offset: 20 });

      await listBooks(10, 20, "tolkien");

      const [url] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books?limit=10&offset=20&query=tolkien");
    });

    it("omits query param when query is empty string", async () => {
      mockFetchResponse({ books: [], total: 0, limit: 50, offset: 0 });

      await listBooks(50, 0, "");

      const [url] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books?limit=50&offset=0");
    });
  });

  describe("getBook", () => {
    it("sends GET /api/books/:id and returns the book", async () => {
      mockFetchResponse(fakeBook);

      const result = await getBook("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeBook);
    });
  });

  describe("createBook", () => {
    it("sends POST /api/books with input body and returns the created book", async () => {
      mockFetchResponse(fakeBook);

      const result = await createBook({ title: "Test Book" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ title: "Test Book" });
      expect(result).toEqual(fakeBook);
    });
  });

  describe("updateBook", () => {
    it("sends PUT /api/books/:id with input body and returns the updated book", async () => {
      const updated: Book = { ...fakeBook, title: "Updated Book" };
      mockFetchResponse(updated);

      const result = await updateBook("b1", { title: "Updated Book" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({ title: "Updated Book" });
      expect(result).toEqual(updated);
    });
  });

  describe("deleteBook", () => {
    it("sends DELETE /api/books/:id", async () => {
      mockNoContentResponse();

      const result = await deleteBook("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });

  describe("getBookAuthors", () => {
    it("sends GET /api/books/:id/authors and returns the authors", async () => {
      mockFetchResponse([fakeAuthor]);

      const result = await getBookAuthors("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/authors");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeAuthor]);
    });
  });

  describe("setBookAuthors", () => {
    it("sends PUT /api/books/:id/authors with author_ids body", async () => {
      mockFetchResponse([fakeAuthor]);

      const result = await setBookAuthors("b1", ["a1"]);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/authors");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({ author_ids: ["a1"] });
      expect(result).toEqual([fakeAuthor]);
    });
  });

  describe("getBookSeries", () => {
    it("sends GET /api/books/:id/series and returns the series entries", async () => {
      mockFetchResponse([fakeSeriesEntry]);

      const result = await getBookSeries("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/series");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeSeriesEntry]);
    });
  });

  describe("setBookSeries", () => {
    it("sends PUT /api/books/:id/series with entries body", async () => {
      mockFetchResponse([fakeSeriesEntry]);

      const result = await setBookSeries("b1", [
        { series_id: "s1", position: 1 },
      ]);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/series");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({
        entries: [{ series_id: "s1", position: 1 }],
      });
      expect(result).toEqual([fakeSeriesEntry]);
    });
  });

  describe("listBookFiles", () => {
    it("sends GET /api/books/:id/files and returns the files", async () => {
      mockFetchResponse([fakeBookFile]);

      const result = await listBookFiles("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/files");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeBookFile]);
    });
  });

  describe("createBookFile", () => {
    it("sends POST /api/books/:id/files with input and returns the created file", async () => {
      mockFetchResponse(fakeBookFile);

      const input = {
        file_type: "epub",
        file_name: "test.epub",
        file_size: 1024,
        file_path: "/books/test.epub",
      };
      const result = await createBookFile("b1", input);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/files");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual(input);
      expect(result).toEqual(fakeBookFile);
    });
  });

  describe("getBookFile", () => {
    it("sends GET /api/book-files/:id and returns the file", async () => {
      mockFetchResponse(fakeBookFile);

      const result = await getBookFile("f1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/book-files/f1");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeBookFile);
    });
  });

  describe("deleteBookFile", () => {
    it("sends DELETE /api/book-files/:id", async () => {
      mockNoContentResponse();

      const result = await deleteBookFile("f1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/book-files/f1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });

  describe("bookFileDownloadUrl", () => {
    it("returns the download URL for a book file", () => {
      const url = bookFileDownloadUrl("f1");
      expect(url).toBe("/api/book-files/f1/download");
    });
  });

  describe("emailBookFile", () => {
    it("sends POST /api/book-files/:id/email with to body and returns the message", async () => {
      mockFetchResponse({ message: "Email sent successfully" });

      const result = await emailBookFile("f1", "reader@example.com");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/book-files/f1/email");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ to: "reader@example.com" });
      expect(result).toEqual({ message: "Email sent successfully" });
    });
  });
});
