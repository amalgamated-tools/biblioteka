import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const STORAGE_KEY = "biblioteka_theme";

// Provide a default matchMedia stub for jsdom which lacks this API.
function makeMatchMedia(prefersDark: boolean) {
  return vi.fn().mockImplementation((query: string) => ({
    matches:
      query === "(prefers-color-scheme: dark)" ? prefersDark : !prefersDark,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }));
}

describe("ThemeStore.set", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
    vi.stubGlobal("matchMedia", makeMatchMedia(false));
    vi.resetModules();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("persists the preference to localStorage", async () => {
    const { themeStore } = await import("./theme.svelte");
    themeStore.set("dark");
    expect(localStorage.getItem(STORAGE_KEY)).toBe("dark");
  });

  it("updates the preference property", async () => {
    const { themeStore } = await import("./theme.svelte");
    themeStore.set("light");
    expect(themeStore.preference).toBe("light");
  });

  it("adds the 'dark' class to <html> when preference is 'dark'", async () => {
    const { themeStore } = await import("./theme.svelte");
    themeStore.set("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("removes the 'dark' class from <html> when preference is 'light'", async () => {
    document.documentElement.classList.add("dark");
    const { themeStore } = await import("./theme.svelte");
    themeStore.set("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("applies dark class when preference is 'auto' and system prefers dark", async () => {
    vi.stubGlobal("matchMedia", makeMatchMedia(true));
    const { themeStore } = await import("./theme.svelte");
    themeStore.set("auto");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("removes dark class when preference is 'auto' and system prefers light", async () => {
    document.documentElement.classList.add("dark");
    vi.stubGlobal("matchMedia", makeMatchMedia(false));
    const { themeStore } = await import("./theme.svelte");
    themeStore.set("auto");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });
});

describe("ThemeStore.init", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
    vi.stubGlobal("matchMedia", makeMatchMedia(false));
    vi.resetModules();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("reads the stored preference and applies it", async () => {
    localStorage.setItem(STORAGE_KEY, "dark");
    vi.stubGlobal("matchMedia", makeMatchMedia(false));
    vi.resetModules();

    const { themeStore } = await import("./theme.svelte");
    themeStore.init();

    expect(themeStore.preference).toBe("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("falls back to 'auto' when no theme is stored", async () => {
    const { themeStore } = await import("./theme.svelte");
    themeStore.init();
    expect(themeStore.preference).toBe("auto");
  });

  it("ignores unknown stored values and defaults to 'auto'", async () => {
    localStorage.setItem(STORAGE_KEY, "system");
    const { themeStore } = await import("./theme.svelte");
    themeStore.init();
    expect(themeStore.preference).toBe("auto");
  });

  it("registers a change listener on the prefers-color-scheme media query", async () => {
    const addEventListenerMock = vi.fn();
    vi.stubGlobal(
      "matchMedia",
      vi.fn().mockReturnValue({
        matches: false,
        addEventListener: addEventListenerMock,
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }),
    );

    const { themeStore } = await import("./theme.svelte");
    themeStore.init();

    expect(addEventListenerMock).toHaveBeenCalledWith(
      "change",
      expect.any(Function),
    );
  });

  it("re-applies theme when system scheme changes and preference is 'auto'", async () => {
    let changeCallback: (() => void) | undefined;
    vi.stubGlobal(
      "matchMedia",
      vi.fn().mockImplementation((query: string) => ({
        matches: query === "(prefers-color-scheme: dark)",
        addEventListener: (_event: string, cb: () => void) => {
          changeCallback = cb;
        },
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    );

    localStorage.setItem(STORAGE_KEY, "auto");

    const { themeStore } = await import("./theme.svelte");
    themeStore.init();

    // Simulate the system switching to light
    vi.stubGlobal(
      "matchMedia",
      vi.fn().mockReturnValue({
        matches: false,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }),
    );

    changeCallback?.();

    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });
});

describe("ThemeStore initial state", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
    vi.stubGlobal("matchMedia", makeMatchMedia(false));
    vi.resetModules();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("reads 'light' preference from localStorage on module load", async () => {
    localStorage.setItem(STORAGE_KEY, "light");
    vi.resetModules();
    const { themeStore } = await import("./theme.svelte");
    expect(themeStore.preference).toBe("light");
  });

  it("reads 'dark' preference from localStorage on module load", async () => {
    localStorage.setItem(STORAGE_KEY, "dark");
    vi.resetModules();
    const { themeStore } = await import("./theme.svelte");
    expect(themeStore.preference).toBe("dark");
  });
});
