import { describe, it, expect, beforeEach, afterEach, vi, type Mock } from "vitest";
import {
  setToken,
  clearToken,
  hasToken,
  ApiError,
  signup,
  login,
  getMe,
  changePassword,
  listArrServices,
  createArrService,
  updateArrService,
  deleteArrService,
  searchMovies,
  listMovies,
  likeMovie,
  unlikeMovie,
  getMovieProviders,
  searchTvSeries,
  listTvSeries,
  likeTvSeries,
  unlikeTvSeries,
  getTvSeriesProviders,
  getConfigStatus,
  setTmdbApiKey,
  getOidcConfig,
  setOidcConfig,
  createOidcLinkNonce,
  listUsers,
  setUserAdmin,
  listWatchProviders,
  getUserWatchProviders,
  setUserWatchProviders,
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

  it("returns undefined for 204 No Content", async () => {
    const resp = {
      ok: true,
      status: 204,
      statusText: "No Content",
      headers: new Headers(),
      json: vi.fn(),
      text: vi.fn(),
    } as unknown as Response;
    (fetchMock).mockResolvedValue(resp);

    const result = await deleteArrService("123");
    expect(result).toBeUndefined();
    expect(resp.json).not.toHaveBeenCalled();
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

describe("Arr Services API", () => {
  beforeEach(() => {
    localStorage.clear();
    fetchMock = vi.fn(); vi.stubGlobal("fetch", fetchMock);
  });

  it("listArrServices calls GET /api/arr-services", async () => {
    mockFetchResponse([{ id: "1", name: "radarr" }]);
    const result = await listArrServices();
    expect(result).toEqual([{ id: "1", name: "radarr" }]);
  });

  it("createArrService sends POST with input", async () => {
    const input = { name: "My Radarr", type: "radarr" as const, url: "http://radarr", api_key: "key" };
    mockFetchResponse({ ...input, id: "1" });

    await createArrService(input);

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/arr-services");
    expect(options.method).toBe("POST");
  });

  it("updateArrService sends PUT with id in path", async () => {
    const input = { name: "Updated", type: "sonarr" as const, url: "http://sonarr", api_key: "key" };
    mockFetchResponse({ ...input, id: "42" });

    await updateArrService("42", input);

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/arr-services/42");
    expect(options.method).toBe("PUT");
  });

  it("deleteArrService sends DELETE with id in path", async () => {
    const resp = {
      ok: true,
      status: 204,
      headers: new Headers(),
      json: vi.fn(),
      text: vi.fn(),
    } as unknown as Response;
    (fetchMock).mockResolvedValue(resp);

    await deleteArrService("42");

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/arr-services/42");
    expect(options.method).toBe("DELETE");
  });
});

describe("Movies API", () => {
  beforeEach(() => {
    localStorage.clear();
    fetchMock = vi.fn(); vi.stubGlobal("fetch", fetchMock);
  });

  it("searchMovies encodes query param", async () => {
    mockFetchResponse([]);
    await searchMovies("the matrix & more");

    const [url] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/movies/search?q=the%20matrix%20%26%20more");
  });

  it("listMovies calls GET /api/movies", async () => {
    mockFetchResponse([{ id: "1", title: "Inception" }]);
    const result = await listMovies();
    expect(result).toEqual([{ id: "1", title: "Inception" }]);
  });

  it("likeMovie sends POST with movie data", async () => {
    const movie = { tmdb_id: 550, title: "Fight Club" };
    mockFetchResponse({ ...movie, id: "1", status: "liked" });

    await likeMovie(movie);

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/movies/550/like");
    expect(options.method).toBe("POST");
  });

  it("unlikeMovie sends DELETE", async () => {
    const resp = {
      ok: true,
      status: 204,
      headers: new Headers(),
      json: vi.fn(),
      text: vi.fn(),
    } as unknown as Response;
    (fetchMock).mockResolvedValue(resp);

    await unlikeMovie(550);

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/movies/550/like");
    expect(options.method).toBe("DELETE");
  });

  it("getMovieProviders calls correct endpoint", async () => {
    mockFetchResponse({ tmdb_id: 550, stream: [], rent: [], buy: [] });
    const result = await getMovieProviders(550);
    expect(result.tmdb_id).toBe(550);
  });
});

describe("TV Series API", () => {
  beforeEach(() => {
    localStorage.clear();
    fetchMock = vi.fn(); vi.stubGlobal("fetch", fetchMock);
  });

  it("searchTvSeries encodes query param", async () => {
    mockFetchResponse([]);
    await searchTvSeries("breaking bad");

    const [url] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/tv-series/search?q=breaking%20bad");
  });

  it("listTvSeries calls GET /api/tv-series", async () => {
    mockFetchResponse([{ id: "1", title: "Breaking Bad" }]);
    const result = await listTvSeries();
    expect(result).toEqual([{ id: "1", title: "Breaking Bad" }]);
  });

  it("likeTvSeries sends POST with series data", async () => {
    const series = { tmdb_id: 1396, title: "Breaking Bad" };
    mockFetchResponse({ ...series, id: "1", status: "liked" });

    await likeTvSeries(series);

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/tv-series/1396/like");
    expect(options.method).toBe("POST");
  });

  it("unlikeTvSeries sends DELETE", async () => {
    const resp = {
      ok: true,
      status: 204,
      headers: new Headers(),
      json: vi.fn(),
      text: vi.fn(),
    } as unknown as Response;
    (fetchMock).mockResolvedValue(resp);

    await unlikeTvSeries(1396);

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/tv-series/1396/like");
    expect(options.method).toBe("DELETE");
  });

  it("getTvSeriesProviders calls correct endpoint", async () => {
    mockFetchResponse({ tmdb_id: 1396, stream: [], buy: [] });
    const result = await getTvSeriesProviders(1396);
    expect(result.tmdb_id).toBe(1396);
  });
});

describe("Config API", () => {
  beforeEach(() => {
    localStorage.clear();
    fetchMock = vi.fn(); vi.stubGlobal("fetch", fetchMock);
  });

  it("getConfigStatus calls GET /api/config/status", async () => {
    mockFetchResponse({ tmdb_configured: true, oidc_configured: false, is_admin: true });
    const result = await getConfigStatus();
    expect(result.tmdb_configured).toBe(true);
    expect(result.is_admin).toBe(true);
  });

  it("setTmdbApiKey sends PUT with api_key", async () => {
    mockFetchResponse({ message: "ok" });
    await setTmdbApiKey("my-tmdb-key");

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/config/tmdb-api-key");
    expect(options.method).toBe("PUT");
    expect(JSON.parse(options.body)).toEqual({ api_key: "my-tmdb-key" });
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

describe("Watch Providers API", () => {
  beforeEach(() => {
    localStorage.clear();
    fetchMock = vi.fn(); vi.stubGlobal("fetch", fetchMock);
  });

  it("listWatchProviders calls GET /api/watch-providers", async () => {
    mockFetchResponse([{ provider_id: 1, provider_name: "Netflix" }]);
    const result = await listWatchProviders();
    expect(result).toHaveLength(1);
  });

  it("getUserWatchProviders calls GET /api/user/watch-providers", async () => {
    mockFetchResponse([]);
    const result = await getUserWatchProviders();
    expect(result).toEqual([]);
  });

  it("setUserWatchProviders sends PUT with provider_ids", async () => {
    mockFetchResponse([{ provider_id: 1 }]);
    await setUserWatchProviders([1, 2, 3]);

    const [url, options] = (fetchMock).mock.calls[0];
    expect(url).toBe("/api/user/watch-providers");
    expect(options.method).toBe("PUT");
    expect(JSON.parse(options.body)).toEqual({ provider_ids: [1, 2, 3] });
  });
});
