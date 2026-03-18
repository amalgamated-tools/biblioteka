export type ThemePreference = "light" | "dark" | "auto";

const STORAGE_KEY = "biblioteka_theme";

function getStoredTheme(): ThemePreference {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "auto")
    return stored;
  return "auto";
}

function getSystemPrefersDark(): boolean {
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

function applyTheme(preference: ThemePreference): void {
  const isDark =
    preference === "dark" || (preference === "auto" && getSystemPrefersDark());

  document.documentElement.classList.toggle("dark", isDark);
}

class ThemeStore {
  preference: ThemePreference = $state(getStoredTheme());

  set(preference: ThemePreference): void {
    this.preference = preference;
    localStorage.setItem(STORAGE_KEY, preference);
    applyTheme(preference);
  }

  init(): void {
    const pref = getStoredTheme();
    this.preference = pref;
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
}

export const themeStore = new ThemeStore();
