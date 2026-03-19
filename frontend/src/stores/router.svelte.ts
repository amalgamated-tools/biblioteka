function getHash(): string {
  const h = window.location.hash.replace(/^#\/?/, "");
  return h || "dashboard";
}

export type AppView =
  | "dashboard"
  | "books"
  | "my-library"
  | "libraries"
  | "settings";

export type SettingsSubPath =
  | "account"
  | "preferences"
  | "oidc"
  | "smtp"
  | "users"
  | "api-keys"
  | "kobo";

const APP_TITLE_SUFFIX = " – biblioteka";

const viewTitles: Record<AppView, string> = {
  dashboard: `Dashboard${APP_TITLE_SUFFIX}`,
  books: `All Books${APP_TITLE_SUFFIX}`,
  "my-library": `My Library${APP_TITLE_SUFFIX}`,
  libraries: `Libraries${APP_TITLE_SUFFIX}`,
  settings: `Settings${APP_TITLE_SUFFIX}`,
};

const settingsSubTitles: Record<SettingsSubPath, string> = {
  account: `Account Settings${APP_TITLE_SUFFIX}`,
  preferences: `Preferences${APP_TITLE_SUFFIX}`,
  oidc: `SSO Settings${APP_TITLE_SUFFIX}`,
  smtp: `Email Settings${APP_TITLE_SUFFIX}`,
  users: `User Management${APP_TITLE_SUFFIX}`,
  "api-keys": `API Keys${APP_TITLE_SUFFIX}`,
  kobo: `Kobo Sync${APP_TITLE_SUFFIX}`,
};

class RouterStore {
  hash = $state(getHash());

  /** Whether the current hash maps to a known view */
  private isKnownView: boolean = $derived.by(() => {
    const segment = this.hash.split("/")[0];
    const valid: AppView[] = [
      "dashboard",
      "books",
      "my-library",
      "libraries",
      "settings",
    ];
    return segment === "" || valid.includes(segment as AppView);
  });

  /** Top-level view derived from hash */
  currentView: AppView = $derived.by(() => {
    const segment = this.hash.split("/")[0];
    const valid: AppView[] = [
      "dashboard",
      "books",
      "my-library",
      "libraries",
      "settings",
    ];
    return valid.includes(segment as AppView)
      ? (segment as AppView)
      : "dashboard";
  });

  /** Sub-path after the top-level segment, e.g. "account" from "settings/account" */
  subPath: string = $derived.by(() => {
    const parts = this.hash.split("/");
    return parts.length > 1 ? parts.slice(1).join("/") : "";
  });

  /** Page title reflecting the current view and settings sub-page */
  pageTitle: string = $derived.by(() => {
    if (!this.isKnownView) return "biblioteka";
    if (this.currentView === "settings" && this.subPath) {
      const subTitle = settingsSubTitles[this.subPath as SettingsSubPath];
      if (subTitle) return subTitle;
    }
    return viewTitles[this.currentView];
  });

  constructor() {
    // Keep store in sync with browser navigation
    if (typeof window !== "undefined") {
      window.addEventListener("hashchange", () => {
        this.hash = getHash();
      });
    }
  }

  /** Navigate by setting the hash */
  navigate(path: string): void {
    window.location.hash = `#${path}`;
    this.hash = path.replace(/^#\/?/, "");
  }
}

export const routerStore = new RouterStore();
