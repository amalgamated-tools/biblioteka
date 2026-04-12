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
  getOpdsCredential,
  setOpdsCredential,
  deleteOpdsCredential,
  getKosyncCredential,
  setKosyncCredential,
  deleteKosyncCredential,
  clearToken,
} from "../api";
import type {
  OpdsCredential,
  OpdsCredentialInput,
  KosyncCredential,
  KosyncCredentialInput,
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

const fakeOpdsCredential: OpdsCredential = {
  username: "opds-user",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-02T00:00:00Z",
};

const fakeOpdsInput: OpdsCredentialInput = {
  username: "opds-user",
  password: "pass1234",
};

const fakeKosyncCredential: KosyncCredential = {
  username: "kosync-user",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-02T00:00:00Z",
};

const fakeKosyncInput: KosyncCredentialInput = {
  username: "kosync-user",
  password: "pass5678",
};

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Credentials API", () => {
  describe("getOpdsCredential", () => {
    it("sends GET /api/opds/credentials and returns the credential", async () => {
      mockFetchResponse(fakeOpdsCredential);

      const result = await getOpdsCredential();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/opds/credentials");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeOpdsCredential);
    });
  });

  describe("setOpdsCredential", () => {
    it("sends PUT /api/opds/credentials with the input body and returns the credential", async () => {
      mockFetchResponse(fakeOpdsCredential);

      const result = await setOpdsCredential(fakeOpdsInput);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/opds/credentials");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual(fakeOpdsInput);
      expect(result).toEqual(fakeOpdsCredential);
    });
  });

  describe("deleteOpdsCredential", () => {
    it("sends DELETE /api/opds/credentials and returns void", async () => {
      mockNoContentResponse();

      const result = await deleteOpdsCredential();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/opds/credentials");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });

  describe("getKosyncCredential", () => {
    it("sends GET /api/kosync/credentials and returns the credential", async () => {
      mockFetchResponse(fakeKosyncCredential);

      const result = await getKosyncCredential();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/kosync/credentials");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeKosyncCredential);
    });
  });

  describe("setKosyncCredential", () => {
    it("sends PUT /api/kosync/credentials with the input body and returns the credential", async () => {
      mockFetchResponse(fakeKosyncCredential);

      const result = await setKosyncCredential(fakeKosyncInput);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/kosync/credentials");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual(fakeKosyncInput);
      expect(result).toEqual(fakeKosyncCredential);
    });
  });

  describe("deleteKosyncCredential", () => {
    it("sends DELETE /api/kosync/credentials and returns void", async () => {
      mockNoContentResponse();

      const result = await deleteKosyncCredential();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/kosync/credentials");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });
});
