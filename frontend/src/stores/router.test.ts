import { describe, it, expect, beforeEach } from "vitest";
import { routerStore } from "./router.svelte";

describe("router store", () => {
  function setHash(h: string) {
    window.location.hash = h;
    window.dispatchEvent(new HashChangeEvent("hashchange"));
  }

  beforeEach(() => {
    setHash("");
  });

  it("defaults to 'dashboard' when hash is empty", () => {
    expect(routerStore.currentView).toBe("dashboard");
    expect(routerStore.subPath).toBe("");
  });

  it("parses 'books' from hash", () => {
    setHash("#books");
    expect(routerStore.currentView).toBe("books");
    expect(routerStore.subPath).toBe("");
  });

  it("parses 'my-library' from hash", () => {
    setHash("#my-library");
    expect(routerStore.currentView).toBe("my-library");
  });

  it("parses 'settings' from hash", () => {
    setHash("#settings");
    expect(routerStore.currentView).toBe("settings");
  });

  it("defaults invalid hash segment to 'dashboard'", () => {
    setHash("#invalid-page");
    expect(routerStore.currentView).toBe("dashboard");
  });

  it("extracts subPath from hash", () => {
    setHash("#settings/account");
    expect(routerStore.currentView).toBe("settings");
    expect(routerStore.subPath).toBe("account");
  });

  it("handles multi-segment subPath", () => {
    setHash("#settings/oidc/config");
    expect(routerStore.subPath).toBe("oidc/config");
  });

  it("navigate sets window.location.hash", () => {
    routerStore.navigate("books");
    expect(window.location.hash).toBe("#books");
  });

  it("responds to hashchange events", () => {
    setHash("#dashboard");

    window.location.hash = "#settings";
    window.dispatchEvent(new HashChangeEvent("hashchange"));

    expect(routerStore.hash).toBe("settings");
  });

  it("handles hash with leading slash", () => {
    setHash("#/books");
    expect(routerStore.currentView).toBe("books");
  });

  describe("isKnownView", () => {
    it("returns true for valid views", () => {
      for (const view of [
        "dashboard",
        "books",
        "my-library",
        "libraries",
        "settings",
      ]) {
        setHash(`#${view}`);
        expect(routerStore.isKnownView).toBe(true);
      }
    });

    it("returns true for empty hash (dashboard default)", () => {
      setHash("");
      expect(routerStore.isKnownView).toBe(true);
    });

    it("returns false for unknown routes", () => {
      setHash("#unknown-page");
      expect(routerStore.isKnownView).toBe(false);
    });

    it("returns false for completely invalid routes", () => {
      setHash("#this-does-not-exist");
      expect(routerStore.isKnownView).toBe(false);
    });
  });

  describe("query parameters", () => {
    it("parses query params from hash", () => {
      setHash("#books?offset=48");
      expect(routerStore.currentView).toBe("books");
      expect(routerStore.queryParams.get("offset")).toBe("48");
    });

    it("parses multiple query params from hash", () => {
      setHash("#books?offset=24&view=table");
      expect(routerStore.queryParams.get("offset")).toBe("24");
      expect(routerStore.queryParams.get("view")).toBe("table");
    });

    it("returns empty URLSearchParams when no query params in hash", () => {
      setHash("#books");
      expect(routerStore.queryParams.get("offset")).toBeNull();
    });

    it("clears query params when navigating to a plain hash", () => {
      setHash("#books?offset=48");
      setHash("#settings");
      expect(routerStore.queryParams.get("offset")).toBeNull();
    });

    it("navigate sets query params from object", () => {
      routerStore.navigate("books", { offset: "48" });
      expect(window.location.hash).toBe("#books?offset=48");
      expect(routerStore.queryParams.get("offset")).toBe("48");
    });

    it("navigate with no params sets no query string", () => {
      routerStore.navigate("books");
      expect(window.location.hash).toBe("#books");
    });

    it("setQueryParam adds a query param to the current hash", () => {
      setHash("#books");
      routerStore.setQueryParam("offset", "24");
      expect(routerStore.queryParams.get("offset")).toBe("24");
    });

    it("setQueryParam removes a query param when value is null", () => {
      setHash("#books?offset=48");
      routerStore.setQueryParam("offset", null);
      expect(routerStore.queryParams.get("offset")).toBeNull();
    });

    it("setQueryParam updates existing query param", () => {
      setHash("#books?offset=24");
      routerStore.setQueryParam("offset", "48");
      expect(routerStore.queryParams.get("offset")).toBe("48");
    });

    it("setQueryParam does not change the current view", () => {
      setHash("#books");
      routerStore.setQueryParam("offset", "24");
      expect(routerStore.currentView).toBe("books");
      expect(routerStore.hash).toBe("books");
    });
  });

  describe("pageTitle", () => {
    it.each([
      ["#dashboard", "Dashboard – biblioteka"],
      ["#books", "All Books – biblioteka"],
      ["#my-library", "My Library – biblioteka"],
      ["#libraries", "Libraries – biblioteka"],
      ["#settings", "Settings – biblioteka"],
      ["#settings/account", "Account Settings – biblioteka"],
      ["#settings/preferences", "Preferences – biblioteka"],
      ["#settings/oidc", "SSO Settings – biblioteka"],
      ["#settings/smtp", "Email Settings – biblioteka"],
      ["#settings/users", "User Management – biblioteka"],
      ["#settings/api-keys", "API Keys – biblioteka"],
    ])("returns '%s' for hash %s", (hash, expected) => {
      setHash(hash);
      expect(routerStore.pageTitle).toBe(expected);
    });

    it("falls back to 'Settings – biblioteka' for unknown settings sub-path", () => {
      setHash("#settings/unknown");
      expect(routerStore.pageTitle).toBe("Settings – biblioteka");
    });

    it("returns 'Page Not Found – biblioteka' for invalid hash", () => {
      setHash("#invalid-page");
      expect(routerStore.pageTitle).toBe("Page Not Found – biblioteka");
    });

    it("preserves page title when hash has query params", () => {
      setHash("#books?offset=48");
      expect(routerStore.pageTitle).toBe("All Books – biblioteka");
    });
  });
});
