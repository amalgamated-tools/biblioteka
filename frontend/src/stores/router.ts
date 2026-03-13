import { writable, derived } from "svelte/store";

/** The raw hash without the leading '#', e.g. "dashboard" */
export const hash = writable(getHash());

function getHash(): string {
  const h = window.location.hash.replace(/^#\/?/, "");
  return h || "dashboard";
}

// Keep store in sync with browser navigation
if (typeof window !== "undefined") {
  window.addEventListener("hashchange", () => {
    hash.set(getHash());
  });
}

/** Navigate by setting the hash */
export function navigate(path: string): void {
  window.location.hash = `#${path}`;
}

export type AppView = "dashboard" | "books" | "my-library" | "settings";

/** Top-level view derived from hash */
export const currentView = derived(hash, ($hash): AppView => {
  const segment = $hash.split("/")[0];
  const valid: AppView[] = ["dashboard", "books", "my-library", "settings"];
  return valid.includes(segment as AppView) ? (segment as AppView) : "dashboard";
});

/** Sub-path after the top-level segment, e.g. "account" from "settings/account" */
export const subPath = derived(hash, ($hash) => {
  const parts = $hash.split("/");
  return parts.length > 1 ? parts.slice(1).join("/") : "";
});
