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
  getToken,
  setToken,
  clearToken,
  hasToken,
  ApiError,
  request,
  getVersion,
} from "./core";

let fetchMock: Mock;

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Token management", () => {
  it("getToken returns null initially", () => {
    expect(getToken()).toBeNull();
  });

  it("setToken stores token and getToken returns it", () => {
    setToken("my-token");
    expect(getToken()).toBe("my-token");
  });

  it("clearToken resets the token to null", () => {
    setToken("my-token");
    clearToken();
    expect(getToken()).toBeNull();
  });

  it("hasToken returns false when no token is set", () => {
    expect(hasToken()).toBe(false);
  });

  it("hasToken returns true when a non-empty token is set", () => {
    setToken("my-token");
    expect(hasToken()).toBe(true);
  });

  it("hasToken returns false when token is an empty string", () => {
    setToken("");
    expect(hasToken()).toBe(false);
  });

  it("hasToken returns false after clearToken", () => {
    setToken("my-token");
    clearToken();
    expect(hasToken()).toBe(false);
  });
});

describe("request()", () => {
  it("returns parsed body on 200 application/json response", async () => {
    const body = { id: "1", name: "test" };
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      statusText: "OK",
      headers: new Headers({ "content-type": "application/json" }),
      json: vi.fn().mockResolvedValue(body),
      text: vi.fn(),
    });

    const result = await request<typeof body>("GET", "/api/test");
    expect(result).toEqual(body);
  });

  it("returns undefined on 204 No Content response", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 204,
      statusText: "No Content",
      headers: new Headers(),
      json: vi.fn(),
      text: vi.fn(),
    });

    const result = await request("DELETE", "/api/test");
    expect(result).toBeUndefined();
  });

  it("falls back to { error: text } on non-JSON content-type", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      statusText: "OK",
      headers: new Headers({ "content-type": "text/plain" }),
      json: vi.fn(),
      text: vi.fn().mockResolvedValue("plain text body"),
    });

    const result = await request<{ error: string }>("GET", "/api/test");
    expect(result).toEqual({ error: "plain text body" });
  });

  it("falls back to {} on JSON parse failure", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      statusText: "OK",
      headers: new Headers({ "content-type": "application/json" }),
      json: vi.fn().mockRejectedValue(new SyntaxError("bad json")),
      text: vi.fn().mockResolvedValue(""),
    });

    const result = await request<Record<string, never>>("GET", "/api/test");
    expect(result).toEqual({});
  });

  it("falls back to { error: text } when JSON parsing fails and body is non-empty", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      statusText: "OK",
      headers: new Headers({ "content-type": "application/json" }),
      json: vi.fn().mockRejectedValue(new SyntaxError("bad json")),
      text: vi.fn().mockResolvedValue("raw error text"),
    });

    const result = await request<{ error: string }>("GET", "/api/test");
    expect(result).toEqual({ error: "raw error text" });
  });

  it("throws ApiError with error message and status on non-ok response", async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 400,
      statusText: "Bad Request",
      headers: new Headers({ "content-type": "application/json" }),
      json: vi.fn().mockResolvedValue({ error: "name is required" }),
      text: vi.fn(),
    });

    await expect(request("POST", "/api/test", {})).rejects.toThrow(ApiError);
    await expect(request("POST", "/api/test", {})).rejects.toMatchObject({
      message: "name is required",
      status: 400,
    });
  });

  it("uses res.statusText when non-ok response has no error field", async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 403,
      statusText: "Forbidden",
      headers: new Headers({ "content-type": "application/json" }),
      json: vi.fn().mockResolvedValue({}),
      text: vi.fn(),
    });

    await expect(request("GET", "/api/test")).rejects.toMatchObject({
      message: "Forbidden",
      status: 403,
    });
  });

  it("sends Authorization header when token is set", async () => {
    setToken("bearer-token");
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      statusText: "OK",
      headers: new Headers({ "content-type": "application/json" }),
      json: vi.fn().mockResolvedValue({}),
      text: vi.fn(),
    });

    await request("GET", "/api/test");

    const [, options] = fetchMock.mock.calls[0];
    expect(options.headers["Authorization"]).toBe("Bearer bearer-token");
  });

  it("does not send Authorization header when no token is set", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      statusText: "OK",
      headers: new Headers({ "content-type": "application/json" }),
      json: vi.fn().mockResolvedValue({}),
      text: vi.fn(),
    });

    await request("GET", "/api/test");

    const [, options] = fetchMock.mock.calls[0];
    expect(options.headers).not.toHaveProperty("Authorization");
  });
});

describe("getVersion()", () => {
  it("returns the version string from the response", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      statusText: "OK",
      headers: new Headers({ "content-type": "application/json" }),
      json: vi.fn().mockResolvedValue({ version: "1.2.3" }),
      text: vi.fn(),
    });

    const version = await getVersion();
    expect(version).toBe("1.2.3");

    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/version");
    expect(options.method).toBe("GET");
  });
});
