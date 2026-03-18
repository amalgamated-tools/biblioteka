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

const viewTitles: Record<AppView, string> = {
  dashboard: "Dashboard – biblioteka",
  books: "All Books – biblioteka",
  "my-library": "My Library – biblioteka",
  libraries: "Libraries – biblioteka",
  settings: "Settings – biblioteka",
};

const settingsSubTitles: Record<string, string> = {
  account: "Account Settings – biblioteka",
  preferences: "Preferences – biblioteka",
  oidc: "SSO Settings – biblioteka",
  smtp: "Email Settings – biblioteka",
  users: "User Management – biblioteka",
  "api-keys": "API Keys – biblioteka",
};

class RouterStore {
  hash = $state(getHash());

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
    if (this.currentView === "settings" && this.subPath) {
      const subTitle = settingsSubTitles[this.subPath];
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
