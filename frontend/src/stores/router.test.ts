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

  describe("pageTitle", () => {
    it("returns 'Dashboard – biblioteka' for dashboard", () => {
      setHash("#dashboard");
      expect(routerStore.pageTitle).toBe("Dashboard – biblioteka");
    });

    it("returns 'All Books – biblioteka' for books", () => {
      setHash("#books");
      expect(routerStore.pageTitle).toBe("All Books – biblioteka");
    });

    it("returns 'My Library – biblioteka' for my-library", () => {
      setHash("#my-library");
      expect(routerStore.pageTitle).toBe("My Library – biblioteka");
    });

    it("returns 'Libraries – biblioteka' for libraries", () => {
      setHash("#libraries");
      expect(routerStore.pageTitle).toBe("Libraries – biblioteka");
    });

    it("returns 'Settings – biblioteka' for settings without sub-path", () => {
      setHash("#settings");
      expect(routerStore.pageTitle).toBe("Settings – biblioteka");
    });

    it("returns 'Account Settings – biblioteka' for settings/account", () => {
      setHash("#settings/account");
      expect(routerStore.pageTitle).toBe("Account Settings – biblioteka");
    });

    it("returns 'Preferences – biblioteka' for settings/preferences", () => {
      setHash("#settings/preferences");
      expect(routerStore.pageTitle).toBe("Preferences – biblioteka");
    });

    it("returns 'SSO Settings – biblioteka' for settings/oidc", () => {
      setHash("#settings/oidc");
      expect(routerStore.pageTitle).toBe("SSO Settings – biblioteka");
    });

    it("returns 'Email Settings – biblioteka' for settings/smtp", () => {
      setHash("#settings/smtp");
      expect(routerStore.pageTitle).toBe("Email Settings – biblioteka");
    });

    it("returns 'User Management – biblioteka' for settings/users", () => {
      setHash("#settings/users");
      expect(routerStore.pageTitle).toBe("User Management – biblioteka");
    });

    it("returns 'API Keys – biblioteka' for settings/api-keys", () => {
      setHash("#settings/api-keys");
      expect(routerStore.pageTitle).toBe("API Keys – biblioteka");
    });

    it("falls back to 'Settings – biblioteka' for unknown settings sub-path", () => {
      setHash("#settings/unknown");
      expect(routerStore.pageTitle).toBe("Settings – biblioteka");
    });

    it("falls back to 'Dashboard – biblioteka' for invalid hash", () => {
      setHash("#invalid-page");
      expect(routerStore.pageTitle).toBe("Dashboard – biblioteka");
    });
  });
});
