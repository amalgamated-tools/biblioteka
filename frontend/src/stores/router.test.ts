import { describe, it, expect, beforeEach } from "vitest";
import { get } from "svelte/store";
import { hash, currentView, subPath, navigate } from "./router";

describe("router store", () => {
  function setHash(h: string) {
    window.location.hash = h;
    window.dispatchEvent(new HashChangeEvent("hashchange"));
  }

  beforeEach(() => {
    setHash("");
  });

  it("defaults to 'dashboard' when hash is empty", () => {
    expect(get(currentView)).toBe("dashboard");
    expect(get(subPath)).toBe("");
  });

  it("parses 'books' from hash", () => {
    setHash("#books");
    expect(get(currentView)).toBe("books");
    expect(get(subPath)).toBe("");
  });

  it("parses 'my-library' from hash", () => {
    setHash("#my-library");
    expect(get(currentView)).toBe("my-library");
  });

  it("parses 'settings' from hash", () => {
    setHash("#settings");
    expect(get(currentView)).toBe("settings");
  });

  it("defaults invalid hash segment to 'dashboard'", () => {
    setHash("#invalid-page");
    expect(get(currentView)).toBe("dashboard");
  });

  it("extracts subPath from hash", () => {
    setHash("#settings/account");
    expect(get(currentView)).toBe("settings");
    expect(get(subPath)).toBe("account");
  });

  it("handles multi-segment subPath", () => {
    setHash("#settings/oidc/config");
    expect(get(subPath)).toBe("oidc/config");
  });

  it("navigate sets window.location.hash", () => {
    navigate("books");
    expect(window.location.hash).toBe("#books");
  });

  it("responds to hashchange events", () => {
    setHash("#dashboard");

    window.location.hash = "#settings";
    window.dispatchEvent(new HashChangeEvent("hashchange"));

    expect(get(hash)).toBe("settings");
  });

  it("handles hash with leading slash", () => {
    setHash("#/books");
    expect(get(currentView)).toBe("books");
  });
});
