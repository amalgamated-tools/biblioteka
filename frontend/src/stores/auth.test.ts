import { describe, it, expect, beforeEach, vi } from "vitest";
import { get } from "svelte/store";
import { user, authLoading, oidcLinkError, initAuth, signIn, signUp, signOut } from "./auth";
import * as api from "../lib/api";

vi.mock("../lib/api", () => ({
  setToken: vi.fn(),
  clearToken: vi.fn(),
  hasToken: vi.fn(),
  getMe: vi.fn(),
  login: vi.fn(),
  signup: vi.fn(),
}));

describe("auth store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    user.set(null);
    authLoading.set(true);
    oidcLinkError.set(null);

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

  describe("initAuth", () => {
    it("sets authLoading to false when no token exists", async () => {
      vi.mocked(api.hasToken).mockReturnValue(false);

      await initAuth();

      expect(get(authLoading)).toBe(false);
      expect(get(user)).toBeNull();
    });

    it("fetches user when token exists", async () => {
      vi.mocked(api.hasToken).mockReturnValue(true);
      vi.mocked(api.getMe).mockResolvedValue({
        id: "1",
        email: "a@b.com",
        oidc_linked: false,
        is_admin: false,
      });

      await initAuth();

      expect(get(user)).toEqual({
        id: "1",
        email: "a@b.com",
        oidc_linked: false,
        is_admin: false,
      });
      expect(get(authLoading)).toBe(false);
    });

    it("clears token and sets user to null when getMe fails", async () => {
      vi.mocked(api.hasToken).mockReturnValue(true);
      vi.mocked(api.getMe).mockRejectedValue(new Error("unauthorized"));

      await initAuth();

      expect(api.clearToken).toHaveBeenCalled();
      expect(get(user)).toBeNull();
      expect(get(authLoading)).toBe(false);
    });

    it("picks up OIDC token from URL params", async () => {
      Object.defineProperty(window, "location", {
        value: {
          search: "?token=oidc-tok",
          pathname: "/",
          hash: "",
        },
        writable: true,
      });
      vi.mocked(api.hasToken).mockReturnValue(true);
      vi.mocked(api.getMe).mockResolvedValue({
        id: "2",
        email: "oidc@b.com",
        oidc_linked: true,
        is_admin: false,
      });

      await initAuth();

      expect(api.setToken).toHaveBeenCalledWith("oidc-tok");
      expect(window.history.replaceState).toHaveBeenCalled();
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

      await initAuth();

      expect(get(oidcLinkError)).toBe("account_already_linked");
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

      await initAuth();

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

      const result = await signIn("a@b.com", "password");

      expect(result.error).toBeNull();
      expect(get(user)).toEqual({
        id: "1",
        email: "a@b.com",
        oidc_linked: false,
        is_admin: false,
      });
    });

    it("returns error on failure and does not set user", async () => {
      vi.mocked(api.login).mockRejectedValue(new Error("bad credentials"));

      const result = await signIn("a@b.com", "wrong");

      expect(result.error).toBeInstanceOf(Error);
      expect(result.error!.message).toBe("bad credentials");
      expect(get(user)).toBeNull();
    });
  });

  describe("signUp", () => {
    it("returns no error on success and sets user", async () => {
      vi.mocked(api.signup).mockResolvedValue({
        token: "tok",
        user: { id: "1", email: "a@b.com", oidc_linked: false, is_admin: true },
      });

      const result = await signUp("Name", "a@b.com", "password");

      expect(result.error).toBeNull();
      expect(get(user)).toEqual({
        id: "1",
        email: "a@b.com",
        oidc_linked: false,
        is_admin: true,
      });
    });

    it("returns error on failure", async () => {
      vi.mocked(api.signup).mockRejectedValue(new Error("email taken"));

      const result = await signUp("Name", "a@b.com", "password");

      expect(result.error).toBeInstanceOf(Error);
      expect(result.error!.message).toBe("email taken");
    });
  });

  describe("signOut", () => {
    it("clears token and user", async () => {
      user.set({ id: "1", email: "a@b.com", oidc_linked: false, is_admin: false });

      await signOut();

      expect(api.clearToken).toHaveBeenCalled();
      expect(get(user)).toBeNull();
    });
  });
});
