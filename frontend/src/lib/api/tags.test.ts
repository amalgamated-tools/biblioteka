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
  listTags,
  getTag,
  createTag,
  updateTag,
  deleteTag,
  clearToken,
} from "../api";
import type { Tag } from "../../types";
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

const fakeTag: Tag = {
  id: "t1",
  name: "science-fiction",
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

describe("Tags API", () => {
  describe("listTags", () => {
    it("sends GET /api/tags and returns the list", async () => {
      mockFetchResponse({ tags: [fakeTag], total: 1, limit: 50, offset: 0 });

      const result = await listTags();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/tags");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeTag]);
    });
  });

  describe("getTag", () => {
    it("sends GET /api/tags/:id and returns the tag", async () => {
      mockFetchResponse(fakeTag);

      const result = await getTag("t1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/tags/t1");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeTag);
    });
  });

  describe("createTag", () => {
    it("sends POST /api/tags with input body and returns the created tag", async () => {
      mockFetchResponse(fakeTag);

      const result = await createTag({ name: "science-fiction" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/tags");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ name: "science-fiction" });
      expect(result).toEqual(fakeTag);
    });
  });

  describe("updateTag", () => {
    it("sends PUT /api/tags/:id with input body and returns the updated tag", async () => {
      const updated: Tag = { ...fakeTag, name: "sci-fi" };
      mockFetchResponse(updated);

      const result = await updateTag("t1", { name: "sci-fi" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/tags/t1");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({ name: "sci-fi" });
      expect(result).toEqual(updated);
    });
  });

  describe("deleteTag", () => {
    it("sends DELETE /api/tags/:id", async () => {
      mockNoContentResponse();

      const result = await deleteTag("t1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/tags/t1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });
});
