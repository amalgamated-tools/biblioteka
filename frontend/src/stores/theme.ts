import { writable } from "svelte/store";

export type ThemePreference = "light" | "dark" | "auto";

const STORAGE_KEY = "biblioteka_theme";

function getStoredTheme(): ThemePreference {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "auto") return stored;
  return "auto";
}

function getSystemPrefersDark(): boolean {
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

export const themePreference = writable<ThemePreference>(getStoredTheme());

function applyTheme(preference: ThemePreference): void {
  const isDark =
    preference === "dark" ||
    (preference === "auto" && getSystemPrefersDark());

  document.documentElement.classList.toggle("dark", isDark);
}

export function setTheme(preference: ThemePreference): void {
  themePreference.set(preference);
  localStorage.setItem(STORAGE_KEY, preference);
  applyTheme(preference);
}

export function initTheme(): void {
  const pref = getStoredTheme();
  themePreference.set(pref);
  applyTheme(pref);

  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", () => {
      const current = getStoredTheme();
      if (current === "auto") {
        applyTheme("auto");
      }
    });
}
