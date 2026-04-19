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
  applyMetadata,
  fetchMetadata,
  getMetadata,
  rejectMetadata,
  subscribeToMetadataEvents,
  getPendingAIEnrichment,
  fetchAIEnrichment,
  applyAIEnrichment,
  rejectAIEnrichment,
  clearToken,
} from "../api";
import type { AIEnrichment, BookSummary, RemoteMetadata } from "../../types";
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

  describe("applyMetadata", () => {
    it("sends POST /api/books/:id/metadata/apply and returns the updated book summary", async () => {
      const fakeBookSummary: BookSummary = {
        id: "b1",
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
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      };
      mockFetchResponse(fakeBookSummary);

      const result = await applyMetadata("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/metadata/apply");
      expect(options.method).toBe("POST");
      expect(result).toEqual(fakeBookSummary);
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

const fakeAIEnrichment: AIEnrichment = {
  id: "ae1",
  book_id: "b1",
  status: "pending",
  provider: "openai",
  model: "gpt-4o-mini",
  suggested_tags: ["fantasy", "adventure"],
  reading_level: "young adult",
  generated_description: "A fantastic journey",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("AI Enrichment API", () => {
  describe("getPendingAIEnrichment", () => {
    it("sends GET /api/books/:id/metadata/ai and returns the enrichment", async () => {
      mockFetchResponse(fakeAIEnrichment);

      const result = await getPendingAIEnrichment("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/metadata/ai");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeAIEnrichment);
    });
  });

  describe("fetchAIEnrichment", () => {
    it("sends POST /api/books/:id/metadata/ai-fetch and returns the response", async () => {
      const response = { task_id: "t2", status: "enqueued" };
      mockFetchResponse(response, 202);

      const result = await fetchAIEnrichment("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/metadata/ai-fetch");
      expect(options.method).toBe("POST");
      expect(result).toEqual(response);
    });
  });

  describe("applyAIEnrichment", () => {
    it("sends POST /api/books/:id/metadata/ai-apply and returns the enrichment", async () => {
      const applied = { ...fakeAIEnrichment, status: "applied" };
      mockFetchResponse(applied);

      const result = await applyAIEnrichment("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/metadata/ai-apply");
      expect(options.method).toBe("POST");
      expect(result).toEqual(applied);
    });
  });

  describe("rejectAIEnrichment", () => {
    it("sends POST /api/books/:id/metadata/ai-reject and returns undefined", async () => {
      mockNoContentResponse();

      const result = await rejectAIEnrichment("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/metadata/ai-reject");
      expect(options.method).toBe("POST");
      expect(result).toBeUndefined();
    });
  });
});
