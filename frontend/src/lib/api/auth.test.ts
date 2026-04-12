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
  login,
  signup,
  logout,
  getMe,
  getOidcEnabled,
  getSignupEnabled,
  changePassword,
  updateProfile,
  createOidcLinkNonce,
  clearToken,
  getToken,
} from "../api";
import type { User } from "../../types";
import { mockFetchResponse as _mockFetchResponse } from "./testUtils";

let fetchMock: Mock;

function mockFetchResponse(body: unknown, status = 200) {
  _mockFetchResponse(fetchMock, body, status);
}

const fakeUser: User = {
  id: "u1",
  name: "Test User",
  email: "test@example.com",
  oidc_linked: false,
  is_admin: false,
};

const fakeAuthResponse = {
  token: "test-token-abc",
  user: fakeUser,
};

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Auth API", () => {
  describe("login", () => {
    it("sends POST /api/auth/login with credentials and calls setToken", async () => {
      mockFetchResponse(fakeAuthResponse);

      const result = await login("test@example.com", "secret");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/auth/login");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({
        email: "test@example.com",
        password: "secret",
      });
      expect(result).toEqual(fakeAuthResponse);
      expect(getToken()).toBe("test-token-abc");
    });
  });

  describe("signup", () => {
    it("sends POST /api/auth/signup with credentials and calls setToken", async () => {
      mockFetchResponse(fakeAuthResponse);

      const result = await signup("Test User", "test@example.com", "secret");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/auth/signup");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({
        name: "Test User",
        email: "test@example.com",
        password: "secret",
      });
      expect(result).toEqual(fakeAuthResponse);
      expect(getToken()).toBe("test-token-abc");
    });
  });

  describe("logout", () => {
    it("sends POST /api/auth/logout", async () => {
      mockFetchResponse({ message: "logged out" });

      await logout();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/auth/logout");
      expect(options.method).toBe("POST");
    });

    it("does not throw when the request fails", async () => {
      mockFetchResponse({ error: "server error" }, 500);

      await expect(logout()).resolves.toBeUndefined();
    });
  });

  describe("getMe", () => {
    it("sends GET /api/auth/me and returns the user", async () => {
      mockFetchResponse(fakeUser);

      const result = await getMe();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/auth/me");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeUser);
    });
  });

  describe("getOidcEnabled", () => {
    it("returns true when the server reports enabled: true", async () => {
      mockFetchResponse({ enabled: true });

      const result = await getOidcEnabled();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/auth/oidc/enabled");
      expect(options.method).toBe("GET");
      expect(result).toBe(true);
    });

    it("returns false when the server reports enabled: false", async () => {
      mockFetchResponse({ enabled: false });

      const result = await getOidcEnabled();

      expect(result).toBe(false);
    });
  });

  describe("getSignupEnabled", () => {
    it("returns true when the server reports enabled: true", async () => {
      mockFetchResponse({ enabled: true });

      const result = await getSignupEnabled();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/auth/signup/enabled");
      expect(options.method).toBe("GET");
      expect(result).toBe(true);
    });

    it("returns false when the server reports enabled: false", async () => {
      mockFetchResponse({ enabled: false });

      const result = await getSignupEnabled();

      expect(result).toBe(false);
    });
  });

  describe("changePassword", () => {
    it("sends PUT /api/auth/password with both password fields", async () => {
      mockFetchResponse({ message: "password changed" });

      await changePassword("old-pass", "new-pass");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/auth/password");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({
        currentPassword: "old-pass",
        newPassword: "new-pass",
      });
    });
  });

  describe("updateProfile", () => {
    it("sends PUT /api/auth/me with name and returns the updated user", async () => {
      const updated: User = { ...fakeUser, name: "New Name" };
      mockFetchResponse(updated);

      const result = await updateProfile("New Name");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/auth/me");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({ name: "New Name" });
      expect(result).toEqual(updated);
    });
  });

  describe("createOidcLinkNonce", () => {
    it("sends POST /api/auth/oidc/link-nonce and returns the nonce string", async () => {
      mockFetchResponse({ nonce: "nonce-xyz-123" });

      const result = await createOidcLinkNonce();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/auth/oidc/link-nonce");
      expect(options.method).toBe("POST");
      expect(result).toBe("nonce-xyz-123");
    });
  });
});
