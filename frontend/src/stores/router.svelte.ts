import { SvelteURLSearchParams } from "svelte/reactivity";

/** Parse the hash into its path and query-parameter components.
 *
 * The hash may optionally carry query parameters after a `?`, e.g.:
 *   `#books?offset=48` → path = "books", params = { offset: "48" }
 */
function parseHash(): { path: string; params: URLSearchParams } {
  const raw = window.location.hash.replace(/^#\/?/, "");
  const qIdx = raw.indexOf("?");
  if (qIdx === -1) {
    return { path: raw || "dashboard", params: new URLSearchParams() };
  }
  return {
    path: raw.slice(0, qIdx) || "dashboard",
    params: new URLSearchParams(raw.slice(qIdx + 1)),
  };
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
  hash = $state(parseHash().path);
  queryParams: SvelteURLSearchParams = $state(
    new SvelteURLSearchParams(parseHash().params.toString()),
  );

  /** Whether the current hash maps to a known view */
  isKnownView: boolean = $derived.by(() => {
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
    if (!this.isKnownView) return `Page Not Found${APP_TITLE_SUFFIX}`;
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
        const { path, params } = parseHash();
        this.hash = path;
        this.queryParams = new SvelteURLSearchParams(params.toString());
      });
    }
  }

  /** Navigate by setting the hash, optionally with query parameters. */
  navigate(path: string, params?: Record<string, string>): void {
    const sp = new SvelteURLSearchParams(params);
    const qs = sp.size > 0 ? `?${sp.toString()}` : "";
    window.location.hash = `#${path}${qs}`;
    this.hash = path.replace(/^#\/?/, "");
    this.queryParams = sp;
  }

  /**
   * Update a single query parameter in the current hash without triggering a
   * navigation (uses `history.replaceState`). Pass `null` to remove the key.
   */
  setQueryParam(key: string, value: string | null): void {
    const newParams = new SvelteURLSearchParams(this.queryParams.toString());
    if (value === null) {
      newParams.delete(key);
    } else {
      newParams.set(key, value);
    }
    const qs = newParams.size > 0 ? `?${newParams.toString()}` : "";
    window.history.replaceState(
      null,
      "",
      `${window.location.pathname}${window.location.search}#${this.hash}${qs}`,
    );
    this.queryParams = newParams;
  }
}

export const routerStore = new RouterStore();
