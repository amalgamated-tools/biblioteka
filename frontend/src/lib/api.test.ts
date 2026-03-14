import { describe, it, expect, beforeEach, afterEach, vi, type Mock } from "vitest";
import {
  setToken,
  clearToken,
  hasToken,
  ApiError,
  signup,
  login,
  getMe,
  getOidcEnabled,
  changePassword,
  getConfigStatus,
  getOidcConfig,
  setOidcConfig,
  createOidcLinkNonce,
  listUsers,
  setUserAdmin,
} from "./api";

const TOKEN_KEY = "biblioteka_token";

let fetchMock: Mock;

afterEach(() => {
  vi.unstubAllGlobals();
});

function mockFetchResponse(
  body: unknown,
  status = 200,
  contentType = "application/json",
) {
  const response = {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? "OK" : "Error",
    headers: new Headers({ "content-type": contentType }),
    json: vi.fn().mockResolvedValue(body),
    text: vi.fn().mockResolvedValue(JSON.stringify(body)),
  } as unknown as Response;

  (fetchMock).mockResolvedValue(response);
  return response;
}

describe("Token management", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("setToken stores token in localStorage", () => {
    setToken("test-token");
    expect(localStorage.getItem(TOKEN_KEY)).toBe("test-token");
  });

  it("clearToken removes token from localStorage", () => {
    localStorage.setItem(TOKEN_KEY, "test-token");
    clearToken();
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
  });

  it("hasToken returns true when token exists", () => {
    localStorage.setItem(TOKEN_KEY, "test-token");
    expect(hasToken()).toBe(true);
  });

  it("hasToken returns false when no token", () => {
    expect(hasToken()).toBe(false);
  });

  it("hasToken returns false for empty string", () => {
    localStorage.setItem(TOKEN_KEY, "");
    expect(hasToken()).toBe(false);
  });
});

describe("ApiError", () => {
  it("creates error with message and status", () => {
    const err = new ApiError("Not Found", 404);
    expect(err.message).toBe("Not Found");
    expect(err.status).toBe(404);
    expect(err.name).toBe("ApiError");
    expect(err).toBeInstanceOf(Error);
  });
});

describe("request (via API functions)", () => {
  beforeEach(() => {
    localStorage.clear();
    fetchMock = vi.fn(); vi.stubGlobal("fetch", fetchMock);
  });

  it("sends Authorization header when token is set", async () => {
    setToken("my-token");
    mockFetchResponse({ id: "1", email: "a@b.com", oidc_linked: false, is_admin: false });

    await getMe();

    expect(fetchMock).toHaveBeenCalledWith("/api/auth/me", {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        Authorization: "Bearer my-token",
      },
      body: undefined,
    });
  });

  it("does not send Authorization header when no token", async () => {
    mockFetchResponse({ id: "1", email: "a@b.com", oidc_linked: false, is_admin: false });

    await getMe();

    expect(fetchMock).toHaveBeenCalledWith("/api/auth/me", {
      method: "GET",
      headers: { "Content-Type": "application/json" },
      body: undefined,
    });
  });

  it("sends JSON body for POST requests", async () => {
    mockFetchResponse({ token: "tok", user: { id: "1", email: "a@b.com" } });

    await signup("Test", "a@b.com", "pass123");

    const [, options] = (fetchMock).mock.calls[0];
    expect(options.method).toBe("POST");
    expect(JSON.parse(options.body)).toEqual({
      name: "Test",
      email: "a@b.com",
      password: "pass123",
    });
  });

  it("throws ApiError on non-OK response with JSON error", async () => {
    const resp = {
      ok: false,
      status: 401,
      statusText: "Unauthorized",
      headers: new Headers({ "content-type": "application/json" }),
      json: vi.fn().mockResolvedValue({ error: "Invalid credentials" }),
      text: vi.fn(),
    } as unknown as Response;
    (fetchMock).mockResolvedValue(resp);

    await expect(login("a@b.com", "wrong")).rejects.toThrow(ApiError);
    await expect(login("a@b.com", "wrong")).rejects.toThrow("Invalid credentials");
  });

  it("throws ApiError on non-OK response with text body", async () => {
    expect.assertions(3);
    const resp = {
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      headers: new Headers({ "content-type": "text/plain" }),
      json: vi.fn(),
      text: vi.fn().mockResolvedValue("something broke"),
    } as unknown as Response;
    (fetchMock).mockResolvedValue(resp);

    try {
      await getMe();
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      expect((e as ApiError).status).toBe(500);
      expect((e as ApiError).message).toBe("something broke");
    }
  });

  it("falls back to statusText when no error message in response", async () => {
    expect.assertions(1);
    const resp = {
      ok: false,
      status: 403,
      statusText: "Forbidden",
      headers: new Headers({ "content-type": "application/json" }),
      json: vi.fn().mockResolvedValue({}),
      text: vi.fn(),
    } as unknown as Response;
    (fetchMock).mockResolvedValue(resp);

    try {
      await getMe();
    } catch (e) {
      expect((e as ApiError).message).toBe("Forbidden");
    }
  });

  it("handles JSON parse failure gracefully", async () => {
    expect.assertions(1);
    const resp = {
      ok: false,
      status: 400,
      statusText: "Bad Request",
      headers: new Headers({ "content-type": "application/json" }),
      json: vi.fn().mockRejectedValue(new SyntaxError("bad json")),
      text: vi.fn().mockResolvedValue("raw error text"),
    } as unknown as Response;
    (fetchMock).mockResolvedValue(resp);

    try {
      await getMe();
    } catch (e) {
      expect((e as ApiError).message).toBe("raw error text");
    }
  });
});

describe("Auth API functions", () => {
  beforeEach(() => {
    localStorage.clear();
    fetchMock = vi.fn(); vi.stubGlobal("fetch", fetchMock);
  });

  it("signup stores token and returns result", async () => {
    const authResp = { token: "new-token", user: { id: "1", email: "a@b.com" } };
    mockFetchResponse(authResp);

    const result = await signup("Name", "a@b.com", "pass");
    expect(result).toEqual(authResp);
    expect(localStorage.getItem(TOKEN_KEY)).toBe("new-token");
  });

  it("login stores token and returns result", async () => {
    const authResp = { token: "login-token", user: { id: "2", email: "b@c.com" } };
    mockFetchResponse(authResp);

    const result = await login("b@c.com", "pass");
    expect(result).toEqual(authResp);
    expect(localStorage.getItem(TOKEN_KEY)).toBe("login-token");
  });

  it("getOidcEnabled returns true when enabled", async () => {
    mockFetchResponse({ enabled: true });

    const result = await getOidcEnabled();
    expect(result).toBe(true);

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/auth/oidc/enabled");
    expect(options.method).toBe("GET");
  });

  it("getOidcEnabled returns false when disabled", async () => {
    mockFetchResponse({ enabled: false });

    const result = await getOidcEnabled();
    expect(result).toBe(false);
  });

  it("getOidcEnabled returns false for unexpected response", async () => {
    mockFetchResponse({});

    const result = await getOidcEnabled();
    expect(result).toBe(false);
  });

  it("changePassword sends PUT request", async () => {
    setToken("tok");
    mockFetchResponse({ message: "ok" });

    await changePassword("old", "new");

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/auth/password");
    expect(options.method).toBe("PUT");
    expect(JSON.parse(options.body)).toEqual({
      currentPassword: "old",
      newPassword: "new",
    });
  });
});

describe("Config API", () => {
  beforeEach(() => {
    localStorage.clear();
    fetchMock = vi.fn(); vi.stubGlobal("fetch", fetchMock);
  });

  it("getConfigStatus calls GET /api/config/status", async () => {
    mockFetchResponse({ oidc_configured: false, is_admin: true });
    const result = await getConfigStatus();
    expect(result.oidc_configured).toBe(false);
    expect(result.is_admin).toBe(true);
  });

  it("getOidcConfig calls GET /api/config/oidc", async () => {
    mockFetchResponse({ issuer_url: "https://issuer", client_id: "id", client_secret_set: false, redirect_uri: "" });
    const result = await getOidcConfig();
    expect(result.issuer_url).toBe("https://issuer");
  });

  it("setOidcConfig sends PUT with config", async () => {
    const config = { issuer_url: "https://issuer", client_id: "id", client_secret: "secret", redirect_uri: "http://redirect" };
    mockFetchResponse({ message: "ok" });
    await setOidcConfig(config);

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/config/oidc");
    expect(options.method).toBe("PUT");
  });

  it("createOidcLinkNonce returns nonce string", async () => {
    mockFetchResponse({ nonce: "abc123" });
    const nonce = await createOidcLinkNonce();
    expect(nonce).toBe("abc123");
  });
});

describe("Admin API", () => {
  beforeEach(() => {
    localStorage.clear();
    fetchMock = vi.fn(); vi.stubGlobal("fetch", fetchMock);
  });

  it("listUsers calls GET /api/admin/users", async () => {
    mockFetchResponse([{ id: "1", name: "Admin", email: "a@b.com", is_admin: true }]);
    const result = await listUsers();
    expect(result).toHaveLength(1);
    expect(result[0].is_admin).toBe(true);
  });

  it("setUserAdmin sends PUT with is_admin", async () => {
    mockFetchResponse({ message: "ok" });
    await setUserAdmin("user-1", true);

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/admin/users/user-1");
    expect(options.method).toBe("PUT");
    expect(JSON.parse(options.body)).toEqual({ is_admin: true });
  });
});
