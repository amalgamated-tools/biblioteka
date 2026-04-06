import { describe, it, expect, beforeEach, afterEach, vi, type Mock } from "vitest";
import {
  listLibraries,
  createLibrary,
  updateLibrary,
  deleteLibrary,
  listLibraryBooks,
} from "../api";
import type { Library, PaginatedBooks } from "../../types";

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

const fakeLibrary: Library = {
  id: "lib1",
  name: "Test Library",
  paths: ["/books"],
  organization_type: "book_per_folder",
  monitored: false,
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

describe("Libraries API", () => {
  describe("listLibraries", () => {
    it("sends GET /api/libraries and returns the list", async () => {
      mockFetchResponse([fakeLibrary]);

      const result = await listLibraries();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/libraries");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeLibrary]);
    });
  });

  describe("createLibrary", () => {
    it("sends POST /api/libraries with input body and returns the created library", async () => {
      mockFetchResponse(fakeLibrary);

      const input = {
        name: "Test Library",
        paths: ["/books"],
        organization_type: "book_per_folder" as const,
        monitored: false,
      };
      const result = await createLibrary(input);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/libraries");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual(input);
      expect(result).toEqual(fakeLibrary);
    });
  });

  describe("updateLibrary", () => {
    it("sends PUT /api/libraries/:id with input body and returns the updated library", async () => {
      const updated: Library = { ...fakeLibrary, name: "Updated Library" };
      mockFetchResponse(updated);

      const input = {
        name: "Updated Library",
        paths: ["/books"],
        organization_type: "book_per_folder" as const,
        monitored: false,
      };
      const result = await updateLibrary("lib1", input);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/libraries/lib1");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual(input);
      expect(result).toEqual(updated);
    });
  });

  describe("deleteLibrary", () => {
    it("sends DELETE /api/libraries/:id", async () => {
      const resp = {
        ok: true,
        status: 204,
        statusText: "No Content",
        headers: new Headers(),
        json: vi.fn(),
        text: vi.fn(),
      } as unknown as Response;
      fetchMock.mockResolvedValue(resp);

      const result = await deleteLibrary("lib1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/libraries/lib1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });

  describe("listLibraryBooks", () => {
    it("sends GET with default limit and offset", async () => {
      const paginatedBooks: PaginatedBooks = {
        books: [],
        total: 0,
        limit: 50,
        offset: 0,
      };
      mockFetchResponse(paginatedBooks);

      const result = await listLibraryBooks("lib1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/libraries/lib1/books?limit=50&offset=0");
      expect(options.method).toBe("GET");
      expect(result).toEqual(paginatedBooks);
    });

    it("passes custom limit and offset", async () => {
      mockFetchResponse({ books: [], total: 10, limit: 10, offset: 20 });

      await listLibraryBooks("lib1", 10, 20);

      const [url] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/libraries/lib1/books?limit=10&offset=20");
    });
  });
});
