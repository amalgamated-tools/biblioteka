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
});
