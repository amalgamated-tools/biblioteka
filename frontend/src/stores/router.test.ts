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

  it("defaults to 'movies' when hash is empty", () => {
    expect(get(currentView)).toBe("movies");
    expect(get(subPath)).toBe("");
  });

  it("parses 'tvshows' from hash", () => {
    setHash("#tvshows");
    expect(get(currentView)).toBe("tvshows");
    expect(get(subPath)).toBe("");
  });

  it("parses 'services' from hash", () => {
    setHash("#services");
    expect(get(currentView)).toBe("services");
  });

  it("parses 'settings' from hash", () => {
    setHash("#settings");
    expect(get(currentView)).toBe("settings");
  });

  it("defaults invalid hash segment to 'movies'", () => {
    setHash("#invalid-page");
    expect(get(currentView)).toBe("movies");
  });

  it("extracts subPath from hash", () => {
    setHash("#movies/browse");
    expect(get(currentView)).toBe("movies");
    expect(get(subPath)).toBe("browse");
  });

  it("handles multi-segment subPath", () => {
    setHash("#settings/oidc/config");
    expect(get(subPath)).toBe("oidc/config");
  });

  it("navigate sets window.location.hash", () => {
    navigate("tvshows");
    expect(window.location.hash).toBe("#tvshows");
  });

  it("responds to hashchange events", () => {
    setHash("#movies");

    window.location.hash = "#settings";
    window.dispatchEvent(new HashChangeEvent("hashchange"));

    expect(get(hash)).toBe("settings");
  });

  it("handles hash with leading slash", () => {
    setHash("#/tvshows");
    expect(get(currentView)).toBe("tvshows");
  });
});
