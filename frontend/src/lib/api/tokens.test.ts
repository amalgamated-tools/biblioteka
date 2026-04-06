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
  listAPIKeys,
  createAPIKey,
  deleteAPIKey,
  listKoboTokens,
  createKoboToken,
  deleteKoboToken,
} from "../api";
import type {
  APIKey,
  APIKeyCreateResponse,
  KoboToken,
  KoboTokenCreateResponse,
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

const fakeAPIKey: APIKey = {
  id: "k1",
  name: "My Key",
  key_prefix: "abc123",
  last_used_at: null,
  created_at: "2026-01-01T00:00:00Z",
};

const fakeAPIKeyCreate: APIKeyCreateResponse = {
  ...fakeAPIKey,
  key: "abc123secrettoken",
};

const fakeKoboToken: KoboToken = {
  id: "kt1",
  user_id: "u1",
  name: "My Kobo",
  created_at: "2026-01-01T00:00:00Z",
};

const fakeKoboTokenCreate: KoboTokenCreateResponse = {
  ...fakeKoboToken,
  token: "kobo-secret-token",
};

beforeEach(() => {
  localStorage.clear();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Tokens API", () => {
  describe("listAPIKeys", () => {
    it("sends GET /api/api-keys and returns the list", async () => {
      mockFetchResponse([fakeAPIKey]);

      const result = await listAPIKeys();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/api-keys");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeAPIKey]);
    });
  });

  describe("createAPIKey", () => {
    it("sends POST /api/api-keys with name and returns the created key", async () => {
      mockFetchResponse(fakeAPIKeyCreate);

      const result = await createAPIKey("My Key");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/api-keys");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ name: "My Key" });
      expect(result).toEqual(fakeAPIKeyCreate);
    });
  });

  describe("deleteAPIKey", () => {
    it("sends DELETE /api/api-keys/:id", async () => {
      mockNoContentResponse();

      const result = await deleteAPIKey("k1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/api-keys/k1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });

  describe("listKoboTokens", () => {
    it("sends GET /api/kobo/tokens and returns the list", async () => {
      mockFetchResponse([fakeKoboToken]);

      const result = await listKoboTokens();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/kobo/tokens");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeKoboToken]);
    });
  });

  describe("createKoboToken", () => {
    it("sends POST /api/kobo/tokens with name and returns the created token", async () => {
      mockFetchResponse(fakeKoboTokenCreate);

      const result = await createKoboToken("My Kobo");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/kobo/tokens");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ name: "My Kobo" });
      expect(result).toEqual(fakeKoboTokenCreate);
    });
  });

  describe("deleteKoboToken", () => {
    it("sends DELETE /api/kobo/tokens/:id", async () => {
      mockNoContentResponse();

      const result = await deleteKoboToken("kt1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/kobo/tokens/kt1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });
});
