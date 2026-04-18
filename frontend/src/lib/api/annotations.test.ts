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
  listBookAnnotations,
  createAnnotation,
  getAnnotation,
  updateAnnotation,
  deleteAnnotation,
  clearToken,
} from "../api";
import type { BookAnnotation } from "../../types";
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

const fakeAnnotation: BookAnnotation = {
  id: "a1",
  user_id: "u1",
  book_id: "b1",
  text: "Great passage",
  cfi: "/p[1]/s[2]",
  user_name: "Alice",
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

describe("Annotations API", () => {
  describe("listBookAnnotations", () => {
    it("sends GET /api/books/:id/annotations and returns the list", async () => {
      mockFetchResponse([fakeAnnotation]);

      const result = await listBookAnnotations("b1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/annotations");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeAnnotation]);
    });
  });

  describe("createAnnotation", () => {
    it("sends POST /api/books/:id/annotations with input body and returns the annotation", async () => {
      mockFetchResponse(fakeAnnotation, 201);

      const result = await createAnnotation("b1", {
        text: "Great passage",
        cfi: "/p[1]/s[2]",
      });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/books/b1/annotations");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({
        text: "Great passage",
        cfi: "/p[1]/s[2]",
      });
      expect(result).toEqual(fakeAnnotation);
    });
  });

  describe("getAnnotation", () => {
    it("sends GET /api/annotations/:id and returns the annotation", async () => {
      mockFetchResponse(fakeAnnotation);

      const result = await getAnnotation("a1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/annotations/a1");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeAnnotation);
    });
  });

  describe("updateAnnotation", () => {
    it("sends PUT /api/annotations/:id with input body and returns the updated annotation", async () => {
      const updated: BookAnnotation = { ...fakeAnnotation, text: "Revised" };
      mockFetchResponse(updated);

      const result = await updateAnnotation("a1", { text: "Revised" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/annotations/a1");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({ text: "Revised" });
      expect(result).toEqual(updated);
    });
  });

  describe("deleteAnnotation", () => {
    it("sends DELETE /api/annotations/:id", async () => {
      mockNoContentResponse();

      const result = await deleteAnnotation("a1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/annotations/a1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });
});
