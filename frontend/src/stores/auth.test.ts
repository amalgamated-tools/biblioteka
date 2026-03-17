import { describe, it, expect, beforeEach, vi } from "vitest";
import { authStore } from "./auth.svelte";
import * as api from "../lib/api";

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../lib/api")>("../lib/api");
  return {
    ApiError: actual.ApiError,
    setToken: vi.fn(),
    clearToken: vi.fn(),
    hasToken: vi.fn(),
    getMe: vi.fn(),
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
  };
});

describe("auth store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authStore.user = null;
    authStore.loading = true;
    authStore.oidcLinkError = null;

    // Reset URL state
    Object.defineProperty(window, "location", {
      value: {
        search: "",
        pathname: "/",
        hash: "",
      },
      writable: true,
    });
    window.history.replaceState = vi.fn();
  });

  describe("init", () => {
    it("sets loading to false when no token and no cookie", async () => {
      vi.mocked(api.hasToken).mockReturnValue(false);
      vi.mocked(api.getMe).mockRejectedValue(
        new api.ApiError("unauthorized", 401),
      );

      await authStore.init();

      expect(authStore.loading).toBe(false);
      expect(authStore.user).toBeNull();
      expect(api.getMe).toHaveBeenCalledTimes(1);
    });

    it("fetches user when token exists", async () => {
      vi.mocked(api.hasToken).mockReturnValue(true);
      vi.mocked(api.getMe).mockResolvedValue({
        id: "1",
        email: "a@b.com",
        oidc_linked: false,
        is_admin: false,
      });

      await authStore.init();

      expect(authStore.user).toEqual({
        id: "1",
        email: "a@b.com",
        oidc_linked: false,
        is_admin: false,
      });
      expect(authStore.loading).toBe(false);
    });

    it("clears token and retries when getMe returns 401 with token", async () => {
      vi.mocked(api.hasToken).mockReturnValue(true);
      vi.mocked(api.getMe).mockRejectedValue(
        new api.ApiError("unauthorized", 401),
      );

      await authStore.init();

      expect(api.clearToken).toHaveBeenCalled();
      expect(api.getMe).toHaveBeenCalledTimes(2);
      expect(authStore.user).toBeNull();
      expect(authStore.loading).toBe(false);
    });

    it("clears stale token and authenticates via cookie on retry", async () => {
      vi.mocked(api.hasToken).mockReturnValue(true);
      vi.mocked(api.getMe)
        .mockRejectedValueOnce(new api.ApiError("unauthorized", 401))
        .mockResolvedValueOnce({
          id: "4",
          email: "oidc@b.com",
          oidc_linked: true,
          is_admin: false,
        });

      await authStore.init();

      expect(api.clearToken).toHaveBeenCalled();
      expect(api.getMe).toHaveBeenCalledTimes(2);
      expect(authStore.user).toEqual({
        id: "4",
        email: "oidc@b.com",
        oidc_linked: true,
        is_admin: false,
      });
      expect(authStore.loading).toBe(false);
    });

    it("preserves token on transient network error", async () => {
      vi.mocked(api.hasToken).mockReturnValue(true);
      vi.mocked(api.getMe).mockRejectedValue(new Error("network error"));

      await authStore.init();

      expect(api.clearToken).not.toHaveBeenCalled();
      expect(api.getMe).toHaveBeenCalledTimes(1);
      expect(authStore.user).toBeNull();
      expect(authStore.loading).toBe(false);
    });

    it("authenticates via cookie on plain reload without token or URL marker", async () => {
      vi.mocked(api.hasToken).mockReturnValue(false);
      vi.mocked(api.getMe).mockResolvedValue({
        id: "3",
        email: "cookie@b.com",
        oidc_linked: true,
        is_admin: false,
      });

      await authStore.init();

      expect(api.getMe).toHaveBeenCalledTimes(1);
      expect(authStore.user).toEqual({
        id: "3",
        email: "cookie@b.com",
        oidc_linked: true,
        is_admin: false,
      });
      expect(authStore.loading).toBe(false);
    });

    it("authenticates via cookie after OIDC redirect", async () => {
      // After OIDC redirect, ?oidc_login=1 is present and no localStorage token,
      // but the HttpOnly cookie is set — getMe() succeeds via cookie.
      Object.defineProperty(window, "location", {
        value: {
          search: "?oidc_login=1",
          pathname: "/",
          hash: "",
        },
        writable: true,
      });
      vi.mocked(api.hasToken).mockReturnValue(false);
      vi.mocked(api.getMe).mockResolvedValue({
        id: "2",
        email: "oidc@b.com",
        oidc_linked: true,
        is_admin: false,
      });

      await authStore.init();

      expect(api.getMe).toHaveBeenCalled();
      expect(authStore.user).toEqual({
        id: "2",
        email: "oidc@b.com",
        oidc_linked: true,
        is_admin: false,
      });
      // URL marker should be cleaned up
      expect(window.history.replaceState).toHaveBeenCalledWith({}, "", "/");
    });

    it("sets oidcLinkError from URL params", async () => {
      Object.defineProperty(window, "location", {
        value: {
          search: "?oidc_link_error=account_already_linked",
          pathname: "/",
          hash: "",
        },
        writable: true,
      });
      vi.mocked(api.hasToken).mockReturnValue(false);
      vi.mocked(api.getMe).mockRejectedValue(new Error("unauthorized"));

      await authStore.init();

      expect(authStore.oidcLinkError).toBe("account_already_linked");
    });

    it("redirects to settings on oidc_linked param", async () => {
      Object.defineProperty(window, "location", {
        value: {
          search: "?oidc_linked=true",
          pathname: "/",
          hash: "",
        },
        writable: true,
      });
      vi.mocked(api.hasToken).mockReturnValue(false);
      vi.mocked(api.getMe).mockRejectedValue(new Error("unauthorized"));

      await authStore.init();

      expect(window.history.replaceState).toHaveBeenCalledWith(
        {},
        "",
        "/#settings",
      );
    });
  });

  describe("signIn", () => {
    it("returns no error on success and sets user", async () => {
      vi.mocked(api.login).mockResolvedValue({
        token: "tok",
        user: { id: "1", email: "a@b.com", oidc_linked: false, is_admin: false },
      });

      const result = await authStore.signIn("a@b.com", "password");

      expect(result.error).toBeNull();
      expect(authStore.user).toEqual({
        id: "1",
        email: "a@b.com",
        oidc_linked: false,
        is_admin: false,
      });
    });

    it("returns error on failure and does not set user", async () => {
      vi.mocked(api.login).mockRejectedValue(new Error("bad credentials"));

      const result = await authStore.signIn("a@b.com", "wrong");

      expect(result.error).toBeInstanceOf(Error);
      expect(result.error!.message).toBe("bad credentials");
      expect(authStore.user).toBeNull();
    });
  });

  describe("signUp", () => {
    it("returns no error on success and sets user", async () => {
      vi.mocked(api.signup).mockResolvedValue({
        token: "tok",
        user: { id: "1", email: "a@b.com", oidc_linked: false, is_admin: true },
      });

      const result = await authStore.signUp("Name", "a@b.com", "password");

      expect(result.error).toBeNull();
      expect(authStore.user).toEqual({
        id: "1",
        email: "a@b.com",
        oidc_linked: false,
        is_admin: true,
      });
    });

    it("returns error on failure", async () => {
      vi.mocked(api.signup).mockRejectedValue(new Error("email taken"));

      const result = await authStore.signUp("Name", "a@b.com", "password");

      expect(result.error).toBeInstanceOf(Error);
      expect(result.error!.message).toBe("email taken");
    });
  });

  describe("signOut", () => {
    it("clears token and user", async () => {
      vi.mocked(api.logout).mockResolvedValue(undefined);
      authStore.user = { id: "1", email: "a@b.com", oidc_linked: false, is_admin: false };

      await authStore.signOut();

      expect(api.clearToken).toHaveBeenCalled();
      expect(api.logout).toHaveBeenCalled();
      expect(authStore.user).toBeNull();
    });
  });
});
