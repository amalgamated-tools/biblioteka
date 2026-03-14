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
  }
}

export const routerStore = new RouterStore();
