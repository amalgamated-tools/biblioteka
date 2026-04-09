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
  fetchMetadata,
  getMetadata,
  rejectMetadata,
  subscribeToMetadataEvents,
  clearToken,
} from "../api";
import type { RemoteMetadata } from "../../types";
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

const fakeMetadata: RemoteMetadata = {
  id: "m1",
  book_id: "b1",
  status: "pending",
  source: "goodreads",
  title: "The Hobbit",
  description: "A fantasy novel",
  asin: null,
  isbn10: null,
  isbn13: "9780547928227",
  goodreads_id: "5907",
  hardcover_id: null,
  google_books_id: null,
  publication_date: "1937-09-21",
  publisher: "Allen & Unwin",
  language: "en",
  cover_image_url: null,
  author_name: "J.R.R. Tolkien",
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

describe("Metadata API", () => {
  describe("fetchMetadata", () => {
    it("sends POST /api/books/:id/metadata/fetch", async () => {
      const response = { task_id: "t1", status: "enqueued" };
      mockFetchResponse(response);

      const result = await fetchMetadata("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/metadata/fetch");
      expect(options.method).toBe("POST");
      expect(result).toEqual(response);
    });
  });

  describe("getMetadata", () => {
    it("sends GET /api/books/:id/metadata and returns the metadata", async () => {
      mockFetchResponse(fakeMetadata);

      const result = await getMetadata("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/metadata");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeMetadata);
    });
  });

  describe("rejectMetadata", () => {
    it("sends POST /api/books/:id/metadata/reject", async () => {
      mockNoContentResponse();

      const result = await rejectMetadata("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/metadata/reject");
      expect(options.method).toBe("POST");
      expect(result).toBeUndefined();
    });
  });

  describe("subscribeToMetadataEvents", () => {
    it("returns an EventSource pointing to the SSE endpoint", () => {
      let capturedUrl = "";
      class MockEventSource {
        url: string;
        close = vi.fn();
        constructor(url: string) {
          this.url = url;
          capturedUrl = url;
        }
      }
      vi.stubGlobal("EventSource", MockEventSource);

      const es = subscribeToMetadataEvents("b1");

      expect(es).toBeInstanceOf(MockEventSource);
      expect(capturedUrl).toBe("/api/books/b1/metadata/events");
    });
  });
});
