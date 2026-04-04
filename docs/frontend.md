# Frontend Architecture

Biblioteka's frontend is a single-page application (SPA) built with **Svelte 5**, **TypeScript**, and **Tailwind CSS 3**. It is bundled by **Vite** and served as static assets embedded in the Go binary.

## Directory layout

```
frontend/
  index.html          HTML entry point; loads favicon, web manifest, and main.ts
  public/             Static assets served at the root URL (copied verbatim by Vite)
    favicon.ico             Default favicon
    favicon-16x16.png       16 × 16 PNG favicon
    favicon-32x32.png       32 × 32 PNG favicon
    apple-touch-icon.png    iOS home-screen icon
    android-chrome-192x192.png  Android home-screen icon (192 × 192)
    android-chrome-512x512.png  Android home-screen icon (512 × 512)
    site.webmanifest        PWA web app manifest (name, icons, theme colour)
  src/
    App.svelte          Root component: auth gate + shell layout + routing; includes skip-to-main-content link (WCAG 2.4.1) and dynamic document title updates (WCAG 2.4.2)
    main.ts             Entry point; mounts App and initialises the theme
    index.css           Tailwind CSS directives
    types.ts            Shared TypeScript interfaces for API entities
    components/         Page-level Svelte components (PascalCase)
      Auth.svelte         Login and signup forms; wraps all form content in a `<main>` landmark (WCAG 1.3.6); the Login/Sign Up toggle uses the ARIA tablist/tab/tabpanel pattern with roving tabindex and keyboard navigation (WCAG 4.1.2)
      Books.svelte        Book listing and detail view; reads `initialOffset` from the URL hash query string (`#books?offset=48`) and writes page changes back via `routerStore.setQueryParam`
      NotFound.svelte     404 page rendered when the router encounters an unknown hash path
      Dashboard.svelte    Home screen; library overview
      Libraries.svelte    Library management view
      MyLibrary.svelte    Placeholder for a planned per-user personal library feature; currently shows an empty state
      Settings.svelte     Settings shell; owns shared admin state; renders one tab at a time
      Sidebar.svelte      Navigation sidebar; fetches and displays the running server version; uses `<a href>` anchor links for all navigation items; the brand name is rendered as `<p>` (not `<h1>`) to avoid duplicate top-level headings (WCAG 1.3.1); icon-only action links (Create library, Library settings) carry `aria-label`, and the Create-library icon explicitly carries `aria-hidden="true"` (WCAG 4.1.2); the Library-settings link aria-label includes the library name (e.g. "Library settings for Fiction") so each link has a unique, descriptive name (WCAG 2.4.6); the Library-settings link always carries at least `opacity-30` so it is visible when focused via keyboard (WCAG 2.4.7); nav link clusters are wrapped in `role="group"` containers labelled by `<h2>` group headings (WCAG 1.3.1)
      libraries/          Reusable sub-components for the Libraries view
        LibraryForm.svelte   Create / edit library form; the "Monitor for new content" toggle uses `role="switch"` and explicit `aria-checked` to communicate on/off state to assistive technologies (WCAG 4.1.2); delete library action uses the `DeleteConfirmation` component for an accessible inline confirmation with keyboard-focus management and Escape-to-dismiss (WCAG 4.1.2)
        LibraryView.svelte   Library detail with book listing
      settings/           Tab sub-components for the Settings page (see Settings component architecture below)
        AccountTab.svelte       Account & password management; OIDC linking
        APIKeysTab.svelte       Create and revoke long-lived API keys (`bib_` prefix); delete actions use an inline `role="alertdialog"` confirmation with keyboard-focus management and Escape-to-dismiss instead of `window.confirm()` (WCAG 4.1.2)
        KoboTab.svelte          Kobo sync token management; displays setup instructions; delete actions use an inline `role="alertdialog"` confirmation with keyboard-focus management and Escape-to-dismiss instead of `window.confirm()` (WCAG 4.1.2)
        OidcTab.svelte          Admin: OIDC / SSO provider configuration
        PreferencesTab.svelte   Display theme selection
        SmtpTab.svelte          Admin: SMTP mail server configuration
        UsersTab.svelte         Admin: user list and admin-role toggling; all `<th>` column headers carry `scope="col"` (WCAG 1.3.1); the role-toggle button carries an action-oriented `aria-label` describing the operation it will perform (WCAG 4.1.2)
      ui/                 Generic reusable UI components
        AlertBanner.svelte   Dismissible alert / error banner
        BookCard.svelte      Card widget displaying a single book summary
        BookList.svelte      Paginated book list with grid / table view toggle; accepts a `fetchBooks` callback; supports optional polling for scan-aware empty states
        Button.svelte        Reusable button with `primary`, `secondary`, and `danger` variants
        DeleteConfirmation.svelte  Accessible inline delete-confirmation dialog (`role="alertdialog"`, Escape-to-dismiss, autofocus on open); encapsulates the standard pattern for accessible destructive-action confirmations (WCAG 4.1.2)
        TextInput.svelte     Reusable text input; forwards all standard `<input>` HTML attributes
    stores/             Reactive state modules (lowercase, *.svelte.ts)
    lib/
      actions.ts              Svelte action utilities (`autofocusFirstButton`)
      api.ts                  Barrel re-export; re-exports every symbol from `api/` sub-modules
      api.test.ts             API client unit tests (tests the barrel and core module)
      api/                    Domain-specific API sub-modules
        core.ts               Token storage, `ApiError`, `request`, `getVersion`
        auth.ts               signup, login, logout, getMe, OIDC, password
        config.ts             OIDC + SMTP server configuration
        admin.ts              User management and audit logs
        credentials.ts        OPDS + KOSync credentials
        libraries.ts          Library CRUD + paginated book listing
        authors.ts            Author CRUD + author–book relationships
        series.ts             Series CRUD + series–book relationships
        books.ts              Book CRUD, associations, and file management
        tokens.ts             API keys, Kobo tokens
      clipboard.ts            Async clipboard helper with `execCommand` fallback
      clipboard.test.ts       Clipboard helper unit tests
      copyTimeout.svelte.ts   `CopyTimeoutState` class — auto-resetting copied-ID feedback state
      copyTimeout.test.ts     Unit tests for `CopyTimeoutState`
      tokenList.svelte.ts     `TokenListState<T>` class — load/delete lifecycle for token-like lists
      tokenList.test.ts       Unit tests for `TokenListState`
      validation.ts           Composable form-validation rule functions
      validation.test.ts      Form-validation unit tests
  vite.config.ts      Vite configuration: build output, dev proxy, Vitest setup, and the restoreGitkeep plugin
```

## Reactive stores

All global state is managed through **Svelte 5 reactive class stores** in `frontend/src/stores/`. Each store is a class whose properties are declared with `$state` (for scalar values and nullable objects) or `$state.raw` (for array properties that are replaced wholesale on fetch), and a singleton instance is exported for use throughout the application.

### Pattern

Most stores that manage a list of entities (authors, series, etc.) extend the generic `CrudStore<T, TInput>` base class defined in `frontend/src/stores/crudStore.svelte.ts`. This base class provides the common `load`, `add`, `edit`, and `remove` operations so individual stores only need to supply the API wiring and any domain-specific accessors.

```ts
// frontend/src/stores/crudStore.svelte.ts — the shared base class
export interface CrudOps<T, TInput> {
  list:   () => Promise<T[]>;
  create: (input: TInput) => Promise<T>;
  update: (id: string, input: TInput) => Promise<T>;
  delete: (id: string) => Promise<void>;
}

export class CrudStore<T extends { id: string }, TInput> {
  items: T[] = $state.raw([]);
  loading = $state(false);
  loaded  = $state(false);

  private readonly ops: CrudOps<T, TInput>;

  constructor(ops: CrudOps<T, TInput>) { this.ops = ops; }

  async load():                             Promise<void> { … }
  async add(input: TInput):                 Promise<T>    { … }
  async edit(id: string, input: TInput):    Promise<T>    { … }
  async remove(id: string):                 Promise<void> { … }
}
```

A concrete store extends `CrudStore` and passes the relevant API functions to the constructor:

```ts
// frontend/src/stores/authors.svelte.ts
import type { Author, AuthorInput } from "../types";
import * as api from "../lib/api";
import { CrudStore } from "./crudStore.svelte";

class AuthorStore extends CrudStore<Author, AuthorInput> {
  constructor() {
    super({
      list:   api.listAuthors,
      create: api.createAuthor,
      update: api.updateAuthor,
      delete: api.deleteAuthor,
    });
  }

  // Named accessor keeps call sites readable: authorStore.authors
  get authors(): Author[] { return this.items; }
  set authors(v: Author[]) { this.items = v; }
}

export const authorStore = new AuthorStore();
```

For stores with additional state beyond basic CRUD (e.g. scan tracking in `libraryStore`), the store may need a fully hand-rolled class instead. `libraryStore` is an example of a store that is currently implemented that way because its `add` flow also maintains scan-tracking state and behavior beyond the base CRUD pattern. If your domain-specific state fits naturally alongside the base CRUD operations, extend `CrudStore` (as `seriesStore` does); otherwise, implement the class directly with `$state` fields.

> **When NOT to extend `CrudStore`**: Use a plain class with `$state` directly for stores that are not entity-list stores (e.g. `authStore`, `routerStore`, `themeStore`).

> **Why classes instead of writable stores?**  
> Svelte 5 introduces fine-grained reactivity via `$state` runes. Using a class groups related state and methods together and makes the reactive surface explicit. It is the idiomatic Svelte 5 approach and replaces the `writable`/`readable` store API from Svelte 4.

> **`$state` vs `$state.raw`**  
> Use `$state` for scalar values (booleans, strings, numbers) and nullable object references (e.g. `user: User | null = $state(null)`). Use `$state.raw` for array properties that are replaced wholesale on every fetch — `$state.raw` tracks only the reference, not the contents of the array. This prevents Svelte from creating a deep reactive proxy over the list items, which avoids unnecessary overhead and eliminates the _"state mutated outside a reactive context"_ console warning that would appear when assigning a new array to a `$state`-annotated property from inside an `async` method.

### Available stores

| File | Export | Purpose |
|------|--------|---------|
| `auth.svelte.ts` | `authStore` | Current user, sign-in/up/out, OIDC token initialisation |
| `router.svelte.ts` | `routerStore` | Hash-based navigation; current view and sub-path |
| `theme.svelte.ts` | `themeStore` | Light / dark / auto theme preference; persisted to `localStorage` |
| `libraries.svelte.ts` | `libraryStore` | Library CRUD; cached after first load; tracks background scan state via `scanningIds` and `isScanning` |
| `authors.svelte.ts` | `authorStore` | Author CRUD; cached after first load |
| `series.svelte.ts` | `seriesStore` | Series CRUD; cached after first load |

### `libraryStore` — scanning state API

When a library is added, the backend scans it asynchronously. `libraryStore` tracks in-progress scans so components can show real-time feedback without polling on their own.

| Member | Type | Description |
|--------|------|-------------|
| `scanningIds` | `SvelteSet<string>` | Reactive set of library IDs whose background scan is in progress. Add a `$derived` check against this set to drive per-library UI. |
| `isScanning` | `boolean` (derived) | `true` when at least one library is currently scanning (`scanningIds.size > 0`). Useful for driving aggregate UI like the all-books view. |
| `clearScanning(id)` | `void` | Removes the given library ID from `scanningIds` and cancels its auto-clear timer. Call this from `BookList`'s `onBooksFound` callback once the scan has completed. |
| `clearAllScanning()` | `void` | Removes all library IDs from `scanningIds` and cancels all timers. Useful in tests or when the user navigates away. |

`add()` automatically marks the newly created library as scanning and schedules an auto-clear after 5 minutes as a safety net (in case the frontend misses the signal). `remove()` calls `clearScanning` for the deleted library so stale scanning state is not left behind.

**Deriving per-library scanning state:**

```svelte
<script lang="ts">
  import { libraryStore } from "../../stores/libraries.svelte";

  let { libraryId }: { libraryId: string } = $props();

  // Reactive: true while this library's scan is in progress
  let scanning = $derived(libraryStore.scanningIds.has(libraryId));
</script>
```

### Using a store in a component

Stores are plain class instances — no special `$` prefix import is needed for their reactive properties when used inside `.svelte` files with the runes compiler:

```svelte
<script lang="ts">
  import { onMount } from "svelte";
  import { libraryStore } from "../stores/libraries.svelte";

  onMount(() => {
    libraryStore.load();
  });
</script>

{#each libraryStore.libraries as lib}
  <p>{lib.name}</p>
{/each}
```

> **Prefer `onMount` for one-time initial data fetching.** `onMount` runs exactly once after the component mounts and is the right place for an unconditional side-effect such as seeding a store. `$effect` re-runs whenever its reactive dependencies change, so using it to trigger `store.load()` can cause repeated fetches or subtle ordering bugs. Only use `$effect` for loading when it is tied to specific reactive preconditions (not just "run on first render") and the `load()` method is idempotent so that repeated calls are safe.

## Routing

Client-side routing uses the browser's URL hash (`#`). No router library is needed.

`routerStore` exposes:

| Property / Method | Type | Description |
|-------------------|------|-------------|
| `hash` | `string` | Raw hash value (e.g. `"books"`) |
| `currentView` | `AppView` | Top-level view segment (`"dashboard"` \| `"books"` \| `"my-library"` \| `"libraries"` \| `"settings"`) |
| `subPath` | `string` | Sub-path after the first segment (e.g. `"account"`) |
| `queryParams` | `SvelteURLSearchParams` | Reactive map of query parameters embedded in the hash (e.g. `?offset=48`). Read with `.get(key)`. |
| `isKnownView` | `boolean` | `true` when `hash` matches a registered view; `false` for unknown routes (triggers the 404 page) |
| `pageTitle` | `string` | Human-readable page title for the current view (e.g. `"Dashboard – biblioteka"`); used to update `document.title` on every navigation |
| `navigate(path, params?)` | `void` | Sets the hash and optionally populates initial query parameters. `params` is a `Record<string, string>`. |
| `setQueryParam(key, value \| null)` | `void` | Updates a single query parameter in the hash via `history.replaceState` without pushing a history entry. Pass `null` to remove the key. |

**Navigating programmatically:**

```ts
import { routerStore } from "../stores/router.svelte";

// Navigate to a view
routerStore.navigate("settings/account");

// Navigate with initial query params
routerStore.navigate("books", { offset: "48" });
```

### 404 handling

When the URL hash does not match any registered view, `routerStore.isKnownView` is `false`. `App.svelte` checks this condition before the view switch and renders `NotFound.svelte` instead of silently falling back to the dashboard. The document title is set to `"Page Not Found – biblioteka"` in this case.

### URL-synced pagination

The `Books` view keeps its pagination offset in the URL hash query string so that the page survives a refresh and can be bookmarked or shared.

- Navigating directly to `#books?offset=48` opens the correct page immediately.
- Turning pages updates the URL in place via `routerStore.setQueryParam("offset", …)` without creating a browser history entry.
- Setting offset back to `0` removes the `offset` parameter from the URL entirely.

**Reading a query parameter:**

```ts
const raw = routerStore.queryParams.get("offset") ?? "0";
const offset = parseInt(raw, 10) || 0;
```

**Writing a query parameter on page change:**

```ts
function handlePageChange(offset: number) {
  routerStore.setQueryParam("offset", offset === 0 ? null : String(offset));
}
```

### Sub-path routing

Views that need their own internal navigation use `routerStore.subPath`. The convention is: `routerStore.hash` holds the normalized fragment without any leading `#/`, it is split on `/`, the first segment becomes the view (`currentView`), and the remaining segments (joined with `/`) form `subPath`, which each view component interprets as it needs.

| View | Sub-path pattern | Meaning |
|------|------------------|---------|
| `libraries` | *(empty)* | List / empty state |
| `libraries` | `new` | Create-library form |
| `libraries` | `{id}` | View a library's books |
| `libraries` | `edit/{id}` | Edit-library form |
| `settings` | `account` | Account settings tab |
| `settings` | `oidc` | OIDC / SSO settings tab |
| `settings` | `smtp` | SMTP mail configuration tab (admin) |
| `settings` | `users` | User management tab (admin) |
| `settings` | `preferences` | Appearance preferences tab |
| `settings` | `api-keys` | API keys management tab (all users) |
| `settings` | `kobo` | Kobo sync token management tab (all users) |

**Example — navigating to a library's book list:**

```ts
// After creating a library, navigate to its detail view
routerStore.navigate(`libraries/${lib.id}`);

// Inside Libraries.svelte, derive the mode from subPath
let mode = $derived.by(() => {
  const sp = routerStore.subPath;
  if (sp === "new")              return "create";
  if (sp.startsWith("edit/"))   return "edit";
  if (sp !== "")                return "view";   // {id} → show books
  return "empty";
});
```

When a view component renders, it reads `routerStore.subPath` as a `$derived` value so it re-renders reactively whenever navigation occurs — no lifecycle hook is needed for the navigation itself.

## API client

All HTTP calls go through `frontend/src/lib/api.ts`, which is a barrel re-export of ten domain-specific sub-modules under `frontend/src/lib/api/`. The barrel preserves all existing import paths so no call-site changes are required when the internal structure evolves.

### Module overview

| Sub-module | Contents |
|------------|----------|
| `api/core.ts` | Token storage (`setToken`, `clearToken`, `hasToken`, `getToken`), `ApiError`, the `request` helper, `getVersion` |
| `api/auth.ts` | `signup`, `login`, `logout`, `getMe`, OIDC helpers, `changePassword` |
| `api/config.ts` | OIDC + SMTP server configuration |
| `api/admin.ts` | User management (`listUsers`, `setUserAdmin`) and audit logs (`getAuditLogs`) |
| `api/credentials.ts` | OPDS and KOSync credential operations |
| `api/libraries.ts` | Library CRUD + paginated book listing |
| `api/authors.ts` | Author CRUD + author–book relationships |
| `api/series.ts` | Series CRUD + series–book relationships |
| `api/books.ts` | Book CRUD, associations, and file management |
| `api/tokens.ts` | API key and Kobo token management |

Each sub-module imports `request` (and `setToken` where needed) from `./core`; there are no circular dependencies.

### Core module

`api/core.ts` contains the building blocks used by all other sub-modules:

- **Token storage** — Stores the JWT in `localStorage` under the key `biblioteka_token`. Use `setToken`, `clearToken`, and `hasToken` to manage it.
- **`ApiError`** — A typed subclass of `Error` that carries a numeric `status` field (the HTTP status code). Catch `ApiError` when you need to branch on a specific status code.
- **`request<T>`** — The shared fetch wrapper. It attaches the `Authorization: Bearer` header when a token is stored and throws `ApiError` on any non-2xx response.

### Usage

Import from the barrel (`../lib/api`) — not from the individual sub-modules — so that call sites do not depend on the internal file layout:

```ts
// Example usage inside a store method
import * as api from "../lib/api";

const book = await api.createBook({ title: "Dune", … });
```

Never call `fetch` directly from components or stores — always go through the API modules.

### `ApiError` handling

```ts
import { ApiError } from "../lib/api";

try {
  await api.login({ username, password });
} catch (err) {
  if (err instanceof ApiError && err.status === 401) {
    // handle wrong credentials
  } else {
    throw err;
  }
}
```

## Utility modules

Several utility modules live in `frontend/src/lib/` alongside `api.ts`.

### `validation.ts`

Composable form-validation helpers. Each exported function returns a `ValidationRule` — a function `(value: string) => string | null` that returns an error message or `null` when the value passes.

| Export | Signature | Description |
|--------|-----------|-------------|
| `ValidationRule` | `type` | A validation rule function |
| `required` | `(message?: string) => ValidationRule` | Fails when the trimmed value is empty |
| `minLength` | `(min: number, message?: string) => ValidationRule` | Fails when the value is shorter than `min` characters |
| `matches` | `(other: string, message?: string) => ValidationRule` | Fails when the value does not equal `other` |
| `validate` | `(value: string, rules: ValidationRule[]) => string \| null` | Runs rules in order; returns the first error or `null` |

**Usage:**

```ts
import { validate, required, minLength, matches } from "../lib/validation";

const passwordError = validate(password, [
  required("Password is required"),
  minLength(8, "Password must be at least 8 characters"),
]);

const confirmError = validate(confirm, [
  required("Please confirm your password"),
  matches(password, "Passwords do not match"),
]);
```

Store the result in a `$state` variable and bind it to a `TextInput` with `aria-invalid` and `aria-describedby` to surface inline errors accessibly (see [Form accessibility](#form-accessibility)).

### `clipboard.ts`

`frontend/src/lib/clipboard.ts` exports a single async function, `copyToClipboard`, that provides a cross-browser interface for writing text to the system clipboard.

```ts
import { copyToClipboard } from "../lib/clipboard";

await copyToClipboard(apiKey);
```

`copyToClipboard` uses the modern async Clipboard API (`navigator.clipboard.writeText`) when available. In environments where the Clipboard API is absent, it falls back to `document.execCommand('copy')` using a hidden `<textarea>`.

> **Note:** The fallback is selected by *availability*, not by failure. If the Clipboard API is present but the browser rejects it (e.g. due to a missing `clipboard-write` permission), the error propagates immediately — the `execCommand` path is not attempted in that case.

It throws an `Error` if the active path fails:
- The Clipboard API rejects (e.g. the user denied the `clipboard-write` permission).
- The `execCommand` fallback returns `false` (the browser blocked the copy command).

> **Guidance:** Always surface copy failures to the user — for example, by displaying an error banner. Do not silently swallow the thrown error.

Use `copyToClipboard` whenever a component needs to copy text (tokens, sync URLs, share links, etc.) to the clipboard. Do **not** inline `navigator.clipboard.writeText` calls in components, as the fallback path would not be covered.

### `tokenList.svelte.ts`

`frontend/src/lib/tokenList.svelte.ts` exports the generic `TokenListState<T extends { id: string }>` class, which manages the load/delete lifecycle for lists of token-like resources (API keys, Kobo tokens, etc.). It encapsulates loading state, inline delete-confirmation flow, and error state using Svelte 5 `$state` runes. The type parameter must include an `id: string` field, because the class uses `item.id` to identify and remove deleted entries from the in-memory list.

**`TokenListOps<T extends { id: string }>` interface** — passed to the constructor:

| Field | Type | Description |
|-------|------|-------------|
| `load` | `() => Promise<T[]>` | Fetches the full list of items; each item must have an `id: string` field |
| `delete` | `(id: string) => Promise<void>` | Deletes the item with the given ID |
| `loadError` | `string` | Fallback error message shown when `load` rejects without an `Error` object |
| `deleteError` | `string` | Fallback error message shown when `delete` rejects without an `Error` object |

**Reactive public fields:**

| Field | Type | Description |
|-------|------|-------------|
| `items` | `T[]` | The loaded list; replaced wholesale on each `load` call and filtered after a successful delete |
| `loading` | `boolean` | `true` while `load()` is in progress |
| `error` | `string \| null` | Load or delete error message; `null` when no error is present |
| `pendingDelete` | `{ id: string; name: string } \| null` | The item currently awaiting confirmation; `null` when no delete is in progress |

**Methods:**

| Method | Description |
|--------|-------------|
| `load()` | Fetches items; sets `loading = true` and clears `error` before the call |
| `handleDelete(id, name)` | Begins the confirmation flow by setting `pendingDelete` |
| `cancelDelete(onAfterClear?)` | Clears `pendingDelete`, waits for a DOM tick, then calls the optional `onAfterClear` callback |
| `cancelDeleteWithFocus()` | Clears `pendingDelete` and returns keyboard focus to the trigger button (`[data-delete-trigger="${id}"]`) |
| `confirmDelete()` | Clears `pendingDelete` immediately (closes the dialog), then calls `ops.delete`; on success removes the item from `items`, on failure sets `error` |

Components that render a delete confirmation dialog should add `data-delete-trigger="${item.id}"` to the Delete button so that `cancelDeleteWithFocus` can restore focus when the user dismisses the dialog.

**Usage:**

```svelte
<script lang="ts">
  import { onMount } from "svelte";
  import { TokenListState } from "../lib/tokenList.svelte";
  import { listAPIKeys, deleteAPIKey } from "../lib/api";
  import type { APIKey } from "../types";

  const tokenList = new TokenListState<APIKey>({
    load: listAPIKeys,
    delete: deleteAPIKey,
    loadError: "Failed to load API keys",
    deleteError: "Failed to delete API key",
  });

  onMount(() => void tokenList.load());
</script>

{#if tokenList.error}
  <AlertBanner variant="error">{tokenList.error}</AlertBanner>
{/if}

{#each tokenList.items as key}
  <button
    data-delete-trigger={key.id}
    onclick={() => tokenList.handleDelete(key.id, key.name)}
  >
    Delete
  </button>
{/each}

{#if tokenList.pendingDelete}
  <div
    role="alertdialog"
    aria-modal="true"
    aria-labelledby={`delete-api-key-title-${tokenList.pendingDelete.id}`}
    tabindex="-1"
    onkeydown={(event) =>
      event.key === "Escape" && tokenList.cancelDeleteWithFocus()}
  >
    <p id={`delete-api-key-title-${tokenList.pendingDelete.id}`}>
      Delete "{tokenList.pendingDelete.name}"?
    </p>
    <button autofocus onclick={() => tokenList.cancelDeleteWithFocus()}>Cancel</button>
    <button onclick={() => tokenList.confirmDelete()}>Confirm</button>
  </div>
{/if}
```

Use `TokenListState` whenever a settings tab manages a list of user-owned tokens. Do **not** re-implement the load/delete/confirmation flow inline in components.

### `copyTimeout.svelte.ts`

`frontend/src/lib/copyTimeout.svelte.ts` exports `CopyTimeoutState`, a small class that manages the "copied" UI feedback state — tracking which item was most recently copied to the clipboard and automatically clearing that state after a configurable duration.

**Constructor:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `duration` | `number` | `2000` | Time in milliseconds before `copiedId` auto-resets to `null` |

**Reactive public fields:**

| Field | Type | Description |
|-------|------|-------------|
| `copiedId` | `string \| null` | ID of the most recently copied item; `null` when no copy feedback is active |

**Methods:**

| Method | Description |
|--------|-------------|
| `set(id)` | Marks `id` as copied, cancels any running timer, and starts a new auto-reset timer |
| `clear()` | Cancels the pending timer and resets `copiedId` to `null` immediately |

Always call `clear()` from `onDestroy` to prevent timer leaks when the component is unmounted before the timeout fires.

**Usage:**

```svelte
<script lang="ts">
  import { onDestroy } from "svelte";
  import { CopyTimeoutState } from "../lib/copyTimeout.svelte";
  import { copyToClipboard } from "../lib/clipboard";

  const copyState = new CopyTimeoutState(); // auto-resets after 2 s
  let copyError: string | null = null;

  onDestroy(() => copyState.clear());

  async function handleCopy(id: string, value: string) {
    try {
      copyError = null;
      await copyToClipboard(value);
      copyState.set(id);
    } catch {
      copyError = "Could not copy to clipboard. Please try again.";
    }
  }
</script>

{#if copyError}
  <p role="alert" class="text-sm text-red-600">{copyError}</p>
{/if}

{#each items as item}
  <button onclick={() => { void handleCopy(item.id, item.token); }}>
    {copyState.copiedId === item.id ? "Copied!" : "Copy"}
  </button>
{/each}
```

Use `CopyTimeoutState` whenever a component shows per-item "Copied!" feedback after a clipboard write. Do **not** manage copy timers inline with `setTimeout` in components.

## TypeScript types

Shared TypeScript interfaces for API entities live in `frontend/src/types.ts`. This includes domain model types (e.g. `Library`, `Author`, `Book`) and shared API request/response shapes (e.g. `ConfigStatus`, `OIDCConfig`, `APIKeyCreateResponse`, `PaginatedAuditLogs`). Keeping shared/exported types in one file gives every component, store, and the API modules a single import path, while individual sub-modules under `frontend/src/lib/api/` may still define small module-local helper types for their own internal use.

Never inline types directly in `.svelte` component files or `*.svelte.ts` store files. Add any new shared or reusable type to `types.ts`.

## Adding a new API function

1. Identify the right sub-module in `frontend/src/lib/api/`. Use the module overview table above as your guide (e.g., a new book endpoint goes in `books.ts`; a new auth endpoint goes in `auth.ts`).
2. Add the exported function to that sub-module. Import `request` from `./core`:

   ```ts
   // frontend/src/lib/api/books.ts
   import { request } from "./core";

   export async function archiveBook(id: string): Promise<void> {
     return request("POST", `/api/books/${id}/archive`);
   }
   ```

3. The barrel (`api.ts`) already re-exports every symbol from every sub-module via `export * from "./api/<module>"`, so the new function is immediately available to all existing import sites — **no changes to `api.ts` are needed**.
4. Add a test in `frontend/src/lib/api.test.ts` covering the new function.

> **Do not add new functions directly to `api.ts`** — that file is a barrel only. Adding logic there breaks the modular structure.

## Adding a new store

1. Create `frontend/src/stores/<name>.svelte.ts`.
2. Define a class with `$state` / `$state.raw` properties. Use `$state.raw` for array properties and `$state` for scalars and nullable objects (see the [`$state` vs `$state.raw`](#reactive-stores) note above).
3. If the store fetches data from the API, implement `load()` with the idempotency guard: `if (this.loading || this.loaded) return;`. This ensures that calling `load()` from multiple `onMount` handlers never issues a duplicate request.
4. Export a singleton: `export const myStore = new MyStore();`.
5. Add an entry for the new store in the table above.

## Adding a new view

1. Create `frontend/src/components/MyView.svelte`.
2. Add the new route identifier to the `AppView` union type in `router.svelte.ts`.
3. Add the route to the `valid` array in `RouterStore.currentView`.
4. Import and render `<MyView />` in `App.svelte` inside the `{#if … }` routing block.
5. Add a navigation entry in `Sidebar.svelte`.

## UI components

The reusable components in `frontend/src/components/ui/` are shared across multiple page-level components. They accept only typed props — no global store access — and are safe to use in any context.

### `AlertBanner.svelte`

Displays a dismissible inline alert in either an error or success style.

**Props:**

| Prop | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `variant` | `"error" \| "success"` | ✓ | — | Visual style and implicit ARIA role |
| `children` | `Snippet` | ✓ | — | Content rendered inside the banner |
| `role` | `string` | | `"alert"` for errors, `"status"` for success | Overrides the implicit ARIA role |
| `testId` | `string` | | — | Sets `data-testid` for test selection |
| `class` | `string` | | — | Additional Tailwind classes appended to the wrapper |

**Usage:**

```svelte
<script lang="ts">
  import AlertBanner from "./ui/AlertBanner.svelte";
  let errorMessage = $state<string | null>(null);
</script>

{#if errorMessage}
  <AlertBanner variant="error">{errorMessage}</AlertBanner>
{/if}
```

The `variant` value controls both the colour scheme and the default ARIA `role`: `"error"` maps to `role="alert"` (announces immediately in screen readers) and `"success"` maps to `role="status"` (polite announcement). Override with the `role` prop when the default is not appropriate.

---

### `Button.svelte`

A styled button with three visual variants.

**Props:**

| Prop | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `variant` | `"primary" \| "secondary" \| "danger"` | | `"primary"` | Visual style |
| `disabled` | `boolean` | | `false` | Disables the button and applies muted styling |
| `type` | `"button" \| "submit" \| "reset"` | | `"button"` | HTML button type |
| `class` | `string` | | — | Additional Tailwind classes (e.g. padding, width) |
| `onclick` | `(e: MouseEvent) => void` | | — | Click handler |
| `children` | `Snippet` | ✓ | — | Button label content |

**Usage:**

```svelte
<script lang="ts">
  import Button from "./ui/Button.svelte";
</script>

<Button variant="primary" type="submit">Save</Button>
<Button variant="secondary" onclick={cancel}>Cancel</Button>
<Button variant="danger" onclick={deleteItem}>Delete</Button>
```

Padding is intentionally left to the caller via the `class` prop to avoid Tailwind cascade conflicts.

---

### `TextInput.svelte`

A styled text input with focus ring, dark-mode support, disabled styling, and ARIA attribute forwarding.

**Props:**

| Prop | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `value` | `string` | | `""` | Bindable input value (`$bindable()`). Supports both `bind:value` and controlled (`value` + `oninput`) patterns. |
| `type` | `"text" \| "email" \| "password" \| "url"` | | `"text"` | Input type — only these four HTML input types are accepted |
| `disabled` | `boolean` | | `false` | Disables the input with muted colors and `cursor-not-allowed` |
| `class` | `string` | | — | Additional Tailwind classes |
| `...restProps` | `HTMLInputAttributes` | | — | Any standard `<input>` attributes (`aria-required`, `aria-invalid`, `aria-describedby`, `placeholder`, `maxlength`, etc.) are forwarded to the underlying element |

**Usage — bind pattern:**

```svelte
<script lang="ts">
  import TextInput from "./ui/TextInput.svelte";
  let name = $state("");
</script>

<TextInput bind:value={name} placeholder="Enter name" aria-required="true" />
```

**Usage — controlled pattern:**

```svelte
<TextInput
  value={name}
  oninput={(e) => (name = e.currentTarget.value)}
  aria-invalid={name.length === 0}
/>
```

---

### `BookCard.svelte`

Renders a single book as a card tile: cover art if available, falling back to a placeholder icon, with the title and publisher below.

**Props:**

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `book` | `BookSummary` | ✓ | The book object to display (imported from `src/types.ts`) |

**Usage:**

```svelte
<script lang="ts">
  import BookCard from "./ui/BookCard.svelte";
  import type { BookSummary } from "../types";

  let book: BookSummary = …;
</script>

<BookCard {book} />
```

`BookCard` is a pure display component — it emits no events and holds no internal state. Use it inside a grid or list layout; `BookList` composes `BookCard` internally.

---

### `BookList.svelte`

A self-contained paginated book browser. It fetches a page of books via a caller-supplied callback, then renders them as either a grid of `BookCard` tiles or a compact table. Navigation between pages is handled internally.

**Props:**

| Prop | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `fetchBooks` | `(limit: number, offset: number) => Promise<PaginatedBooks>` | ✓ | — | Called whenever the page or page size changes. Use `api.listBooks` or `api.listLibraryBooks` as the value. |
| `pageSize` | `number` | | `24` | Number of books per page. Clamped to `[1, 200]` at runtime. |
| `initialOffset` | `number` | | `0` | Starting page offset read **once** on mount. Use this to restore a bookmarked page (e.g., from the URL query string). Changes after mount are ignored. |
| `onPageChange` | `(offset: number) => void` | | — | Called after each page turn (not on the initial mount). Use this to write the new offset back to the URL via `routerStore.setQueryParam`. |
| `pollingInterval` | `number` | | — | When set, `BookList` re-fetches silently at this interval (in ms) while `total === 0`. Polling stops automatically once books appear. Use this to show a "Scanning library..." spinner while the backend scans a newly added library. |
| `onBooksFound` | `() => void` | | — | Called exactly once the first time a poll or fetch reports `total > 0`. Use this to clear scanning state in the parent (e.g., `() => libraryStore.clearScanning(libraryId)`). |

**Internal state exposed to the template (not props):**

| Name | Type | Description |
|------|------|-------------|
| `books` | `BookSummary[]` | Current page of book objects |
| `total` | `number` | Total books across all pages (from the API) |
| `loading` | `boolean` | `true` while a fetch is in flight |
| `error` | `string \| null` | Error message from the most recent failed fetch |
| `viewMode` | `"grid" \| "table"` | User-selected display mode; toggle buttons are rendered by the component |

**Usage — all books:**

```svelte
<script lang="ts">
  import BookList from "./ui/BookList.svelte";
  import * as api from "../lib/api";
</script>

<BookList fetchBooks={api.listBooks} />
```

**Usage — books within a specific library:**

```svelte
<script lang="ts">
  import BookList from "../ui/BookList.svelte";
  import * as api from "../../lib/api";

  let { libraryId }: { libraryId: string } = $props();
</script>

<BookList fetchBooks={(limit, offset) => api.listLibraryBooks(libraryId, limit, offset)} />
```

> **Tip:** When binding a library-scoped `fetchBooks`, wrap `api.listLibraryBooks` in an arrow function so that `libraryId` is captured by the closure. `BookList` resets to page 1 whenever the `fetchBooks` prop reference changes, so switching libraries automatically resets pagination.

#### Pagination behaviour

- On mount and whenever `fetchBooks` or `pageSize` changes, `offset` resets to `0` and a fresh fetch is triggered.
- If items are deleted and the current page becomes empty (but earlier pages still have items), `BookList` automatically clamps back to the last valid page.
- Stale responses from superseded fetches are silently discarded via an internal request-ID counter.

#### Scan-aware polling

When `pollingInterval` is set and `total === 0`, `BookList` enters a polling mode: instead of showing the generic "No books yet." empty state, it renders a spinner with the message "Scanning library...". A `setTimeout`-based loop (not `setInterval`) re-fetches silently at the given interval, suppressing the loading overlay so the spinner stays visible. Polling stops as soon as `total > 0` or the component unmounts. On the first fetch that reports `total > 0`, `onBooksFound` fires exactly once, giving the parent an opportunity to clear scanning state (see `libraryStore.clearScanning` below).

```svelte
<script lang="ts">
  import BookList from "../ui/BookList.svelte";
  import * as api from "../../lib/api";
  import { libraryStore } from "../../stores/libraries.svelte";

  let { libraryId }: { libraryId: string } = $props();

  let scanning = $derived(libraryStore.scanningIds.has(libraryId));
</script>

<BookList
  fetchBooks={(limit, offset) => api.listLibraryBooks(libraryId, limit, offset)}
  pollingInterval={scanning ? 3000 : undefined}
  onBooksFound={scanning ? () => libraryStore.clearScanning(libraryId) : undefined}
/>
```

> **Note:** `BookList` cannot identify *which* library finished scanning, so when embedding the all-books view (`Books.svelte`) alongside a scan in progress, pass `pollingInterval` derived from `libraryStore.isScanning` but **omit** `onBooksFound` — the aggregate view cannot safely call `clearScanning` without knowing the specific library ID. Polling stops naturally once `total > 0`.

---

### `Button.svelte`

A styled button with three visual variants. Use this instead of a raw `<button>` element whenever a button appears in the application UI, so styling is consistent.

**Props:**

| Prop | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `variant` | `"primary" \| "secondary" \| "danger"` | | `"primary"` | Visual style |
| `type` | `"button" \| "submit" \| "reset"` | | `"button"` | HTML button type |
| `disabled` | `boolean` | | `false` | Disables the button and applies reduced-opacity styling |
| `class` | `string` | | — | Additional Tailwind classes appended to the button |
| `onclick` | `(e: MouseEvent) => void` | | — | Click handler |
| `children` | `Snippet` | ✓ | — | Button label rendered as slot content |

**Variants:**

| Variant | When to use |
|---------|-------------|
| `primary` | Primary call-to-action; accent-colored gradient background |
| `secondary` | Secondary action; transparent background with a subtle border |
| `danger` | Destructive actions such as delete or revoke; red background |

**Usage:**

```svelte
<script lang="ts">
  import Button from "./ui/Button.svelte";

  function handleSave() { … }
  function handleCancel() { … }
  function handleDelete() { … }
</script>

<Button onclick={handleSave}>Save</Button>
<Button variant="secondary" onclick={handleCancel}>Cancel</Button>
<Button variant="danger" onclick={handleDelete}>Delete</Button>
```

Padding is intentionally left to the caller via the `class` prop to avoid Tailwind cascade conflicts.

---

### `DeleteConfirmation.svelte`

An accessible inline delete-confirmation dialog that replaces the current item's Delete button with a two-button (`Delete` / `Cancel`) confirmation row. Use this whenever a destructive action requires a user confirmation step. It implements the full accessible pattern — `role="alertdialog"`, autofocus on open, and Escape-to-dismiss — so consumers do not need to repeat any of that boilerplate.

**Props:**

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `itemId` | `string` | ✓ | Unique ID of the item being deleted. Used to generate a stable `aria-labelledby` value. |
| `itemName` | `string` | ✓ | Human-readable name shown in the `"Delete "{itemName}"?"` label. |
| `onConfirm` | `() => void` | ✓ | Called when the user clicks **Delete**. |
| `onCancel` | `() => void` | ✓ | Called when the user clicks **Cancel** or presses Escape. Should restore focus to the original trigger button. |
| `class` | `string` | | Additional Tailwind classes appended to the wrapper. |

**Behavior:**

- Renders a `role="alertdialog"` container (not `role="dialog"`) so screen readers announce its content immediately.
- Uses `use:autofocusFirstButton` to move keyboard focus into the dialog when it mounts.
- An `onkeydown` Escape handler calls `onCancel` to dismiss the dialog.

**Usage:**

```svelte
<script lang="ts">
  import { tick } from "svelte";
  import DeleteConfirmation from "./ui/DeleteConfirmation.svelte";

  // These come from your component's props or local state:
  const itemId = "item-123";
  const item = { name: "Example item" };
  const api = {
    async deleteItem(id: string) { /* your delete logic */ }
  };

  let showDeleteConfirm = $state(false);
  let deleteButtonEl: HTMLButtonElement | null = $state(null);

  async function handleDelete() {
    await api.deleteItem(itemId);
    showDeleteConfirm = false;
  }

  async function cancelDelete() {
    showDeleteConfirm = false;
    await tick(); // wait for Svelte to re-mount the trigger button
    deleteButtonEl?.focus(); // restore focus to the trigger
  }
</script>

{#if showDeleteConfirm}
  <DeleteConfirmation
    {itemId}
    itemName={item.name}
    onConfirm={handleDelete}
    onCancel={cancelDelete}
  />
{:else}
  <button
    bind:this={deleteButtonEl}
    data-delete-trigger={itemId}
    onclick={() => (showDeleteConfirm = true)}
  >Delete</button>
{/if}
```

`LibraryForm.svelte`, `APIKeysTab.svelte`, and `KoboTab.svelte` are canonical reference implementations.

---

### `TextInput.svelte`

A styled text input that forwards all standard HTML `<input>` attributes. Use this instead of a raw `<input>` element so focus-ring, border, dark-mode, and disabled styles are consistent.

**Props:**

| Prop | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `value` | `string` | | `""` | Bindable current value (`$bindable()`). Supports both `bind:value` and controlled (`value` + `oninput`) patterns. |
| `type` | `"text" \| "email" \| "password" \| "url"` | | `"text"` | Input type — only these four HTML input types are accepted. |
| `disabled` | `boolean` | | `false` | Disables the input with muted colors and `cursor-not-allowed` |
| `class` | `string` | | — | Additional Tailwind classes |
| `...restProps` | `HTMLInputAttributes` | | — | Any standard `<input>` attributes (`aria-required`, `aria-invalid`, `aria-describedby`, `placeholder`, `maxlength`, etc.) are forwarded to the underlying element |

**Usage — controlled (bind):**

```svelte
<script lang="ts">
  import TextInput from "./ui/TextInput.svelte";
  let email = $state("");
</script>

<TextInput bind:value={email} type="email" placeholder="you@example.com" />
```

**Usage — uncontrolled (no initial value needed):**

```svelte
<TextInput type="password" aria-label="Password" />
```

For inline validation errors, pass `aria-invalid` and `aria-describedby` through `restProps` and wire them to your validation state (see [Form accessibility](#form-accessibility)).

---

## Settings component architecture

`Settings.svelte` is a shell that owns shared state (admin flag, OIDC config) and renders one tab at a time. Each tab is a standalone sub-component in `frontend/src/components/settings/`.

| Component | Route | Visibility | Responsibility |
|-----------|-------|------------|----------------|
| `AccountTab.svelte` | `settings/account` | All users | Change password; link OIDC account |
| `APIKeysTab.svelte` | `settings/api-keys` | All users | Create and revoke long-lived API keys (`bib_` prefix); uses inline `role="alertdialog"` confirmations for delete actions |
| `KoboTab.svelte` | `settings/kobo` | All users | Create and revoke Kobo sync tokens; copy device sync URL; uses inline `role="alertdialog"` confirmations for delete actions |
| `PreferencesTab.svelte` | `settings/preferences` | All users | Choose light / dark / auto theme |
| `OidcTab.svelte` | `settings/oidc` | Admins only | Configure OIDC / SSO provider |
| `SmtpTab.svelte` | `settings/smtp` | Admins only | Configure SMTP mail server |
| `UsersTab.svelte` | `settings/users` | Admins only | List users; toggle admin role |

`Settings.svelte` passes data down as props and receives updates via callback props (`onOidcSaved`, `onUsersLoaded`), keeping each tab stateless with respect to shared data.

### SmtpTab (`settings/smtp`)

`SmtpTab.svelte` renders a form for configuring the outgoing mail server. It is only shown to admin users.

**Form fields:**

| Field | Required | Notes |
|-------|----------|-------|
| Host | Yes | Hostname or IP of the SMTP server |
| Port | Yes | Defaults to `587` |
| TLS Mode | Yes | Dropdown: `STARTTLS` (default), `TLS`, or `None`. Authenticated SMTP without TLS is blocked for non-loopback servers |
| Username | No | Leave empty for unauthenticated relay |
| Password | Conditional | Required when Username is set. Leave blank on update to keep the existing password |
| From Address | Yes | Envelope `From` address; must be a valid email |

**Key behaviours:**

- **Status badge** — Shows *Configured* (green) or *Not configured* (grey) based on whether a complete SMTP config exists in the database.
- **Test Email button** — Visible only when the server is configured. Sends a test message to the authenticated user's email address. Subject to rate-limiting (one send per minute).
- **Environment-variable override banner** — When `SMTP_HOST` is set as a server environment variable, a blue informational banner replaces the form: *"SMTP is configured via environment variables and cannot be changed here."* The tab becomes read-only. To use the UI instead, unset `SMTP_HOST` from the environment and restart the server.
- **Password preservation** — On save, leaving the password field blank preserves the existing stored credential; fill it only to change it.

### One-time prop initialisation (`svelte-ignore state_referenced_locally`)

Some settings tabs receive initial values from `Settings.svelte` as props and then manage those values as **local state** for the duration of the tab's lifetime. Because the values are not expected to react to future prop changes (the parent passes them once at mount), the tabs use `$state(initialProp)` to seed local state:

```svelte
<script lang="ts">
  interface Props {
    initialIssuerUrl?: string;
    // …
  }
  let { initialIssuerUrl = "" }: Props = $props();

  // One-time initialisation – this prop is not expected to change after mount.
  // svelte-ignore state_referenced_locally
  let issuerUrl = $state(initialIssuerUrl);
</script>
```

Svelte 5 emits a `state_referenced_locally` warning for this pattern because the resulting `$state` variable does not track updates to the prop — it captures the value at creation time only. The `// svelte-ignore state_referenced_locally` comment suppresses the warning when this one-time seeding is **intentional**. Do not use this suppression for state that should reactively follow a prop; use `$derived` or `$effect` instead.

### Adding a new settings tab

1. Create `frontend/src/components/settings/MyTab.svelte`.
2. Define an `interface Props { … }` and use `$props()` for any data the tab needs from `Settings.svelte`.
3. Add `"my-tab"` to the `SettingsTab` union type and `validTabs` array in `Settings.svelte`.
4. Import and render `<MyTab />` inside the `{#if activeTab === "my-tab"}` block in `Settings.svelte`.
5. Add a navigation `<a href="#settings/my-tab">` link in `Settings.svelte`'s sidebar `<nav>`, wrapped in `{#if isAdmin}` if the tab is admin-only.
6. Add `"my-tab"` to the `SettingsSubPath` union type **and** the `settingsSubTitles` record in `frontend/src/stores/router.svelte.ts`. This ensures the browser tab title is set correctly (e.g., `My Tab – biblioteka`). If you skip this step, the title falls back to the top-level `Settings – biblioteka`.
7. Update the tables above and in the [Page titles](#page-titles) section.

## Accessibility Patterns

Biblioteka's frontend follows [WCAG 2.1](https://www.w3.org/TR/WCAG21/) guidelines. This section documents the accessibility patterns used across the app and how to maintain them when making changes.

### Skip-to-main-content link

**WCAG criterion:** [2.4.1 Bypass Blocks](https://www.w3.org/WAI/WCAG21/Understanding/bypass-blocks.html) (Level A)

The authenticated app shell (`App.svelte`) includes a skip link as the **first focusable element** in the DOM. It allows keyboard and screen-reader users to jump past the persistent navigation sidebar directly to the main content area without tabbing through every nav item on every page load.

**Implementation in `App.svelte`:**

```svelte
<!-- First focusable element inside the authenticated shell -->
<a
  href="#main-content"
  onclick={(e: MouseEvent) => {
    e.preventDefault();
    document.getElementById("main-content")?.focus();
  }}
  class="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[100]
         focus:rounded-xl focus:bg-accent-600 focus:px-4 focus:py-2 focus:font-semibold focus:text-white"
>
  Skip to main content
</a>

<!-- … Sidebar … -->

<main id="main-content" tabindex="-1" class="md:ml-64 p-4 md:p-8">
  <!-- page content -->
</main>
```

Key details:

| Element | Attribute / class | Purpose |
|---------|-------------------|---------|
| `<a href="#main-content">` | — | Standard skip link; navigates to the `main` landmark |
| `<a>` | `onclick` handler | Calls `focus()` programmatically so the browser moves keyboard focus, not just scroll position |
| `<a>` | `sr-only focus:not-sr-only` | Visually hidden at rest; fully visible when focused — keeps the default chrome clean while remaining reachable by keyboard |
| `<main>` | `id="main-content"` | Stable anchor target for the skip link |
| `<main>` | `tabindex="-1"` | Allows `element.focus()` to land on the `<main>` element even though it is not natively focusable |

**DOM ordering rule:** The skip link must be rendered **before** `<Sidebar />` in the template so it is the first element reached by the Tab key. Do not move it below the sidebar.

### Page title on navigation

**WCAG criterion:** [2.4.2 Page Titled](https://www.w3.org/WAI/WCAG21/Understanding/page-titled.html) (Level A)

In a single-page application the browser does not perform a real page load on navigation, so `document.title` stays unchanged unless the application updates it explicitly. Screen readers and browser history both rely on meaningful, descriptive page titles to help users understand where they are.

`routerStore` exposes a reactive `pageTitle` property derived from the current view and settings sub-path. `App.svelte` writes it to `document.title` via a Svelte `$effect`:

```svelte
<!-- App.svelte -->
$effect(() => {
  document.title = routerStore.pageTitle;
});
```

`pageTitle` is built from two lookup tables defined in `router.svelte.ts`:

| View / sub-path | Title |
|-----------------|-------|
| `dashboard` | `Dashboard – biblioteka` |
| `books` | `All Books – biblioteka` |
| `my-library` | `My Library – biblioteka` |
| `libraries` | `Libraries – biblioteka` |
| `settings` (no sub-path) | `Settings – biblioteka` |
| `settings/account` | `Account Settings – biblioteka` |
| `settings/preferences` | `Preferences – biblioteka` |
| `settings/oidc` | `SSO Settings – biblioteka` |
| `settings/smtp` | `Email Settings – biblioteka` |
| `settings/users` | `User Management – biblioteka` |
| `settings/api-keys` | `API Keys – biblioteka` |
| `settings/kobo` | `Kobo Sync – biblioteka` |
| Unknown hash | `biblioteka` |

**When adding a new view or settings tab**, update both the corresponding union type (`AppView` or `SettingsSubPath`) and the matching title lookup table in `router.svelte.ts`. If you skip the lookup entry, `pageTitle` falls back to the top-level view title, which may be insufficiently descriptive.

### Focus management on SPA navigation

**WCAG criterion:** [2.4.3 Focus Order](https://www.w3.org/WAI/WCAG21/Understanding/focus-order.html) (Level A)

In a standard multi-page website the browser automatically returns keyboard focus to the top of the document after each page load. Single-page applications do not perform real page loads on navigation, so without explicit focus management keyboard-only and screen-reader users remain focused on whichever element they last interacted with — typically a sidebar navigation link — instead of landing on the new page content.

`App.svelte` addresses this by programmatically moving keyboard focus to `<main id="main-content">` after each route change:

```svelte
<!-- App.svelte -->
let focusEffectMounted = false;
$effect(() => {
  void routerStore.hash; // register hash as a reactive dependency
  if (!focusEffectMounted) {
    focusEffectMounted = true;
    return; // skip initial mount — avoids stealing focus from browser UI on hard refresh
  }
  if (authStore.user) {
    void tick().then(() => {
      document.getElementById("main-content")?.focus();
    });
  }
});
```

Key details:

| Detail | Purpose |
|--------|---------|
| `void routerStore.hash` | Registers `routerStore.hash` as a reactive dependency so the effect re-runs on every navigation |
| `focusEffectMounted` guard | Skips the very first execution so a hard refresh does not steal focus from the browser address bar or other browser UI |
| `tick().then(…)` | Defers `focus()` until after Svelte has flushed DOM updates for the incoming view, so the element is in its final state when focus lands |
| `tabindex="-1"` on `<main>` | Makes `<main>` programmatically focusable even though it is not a natively interactive element |

**When adding a new view,** no additional changes are required — the focus effect fires automatically whenever `routerStore.hash` changes, regardless of which view component is rendered.

### Focus visible — Library settings link (`Sidebar.svelte`)

**WCAG criteria:**
- [2.4.7 Focus Visible](https://www.w3.org/WAI/WCAG21/Understanding/focus-visible.html) (Level AA) — the opacity fix
- [2.4.6 Headings and Labels](https://www.w3.org/WAI/WCAG21/Understanding/headings-and-labels.html) (Level AA) — the unique `aria-label` per library

Interactive elements must have a visible focus indicator so keyboard users can tell which element currently has focus. The library-settings gear icon link in the sidebar is a subtle secondary action that should not visually dominate the sidebar. Before this fix it used `opacity-0` on its resting state, which made it invisible when focused via the keyboard — a WCAG 2.4.7 violation.

The link now uses `opacity-30` as its resting opacity and `opacity-100` on hover or when the parent library row is focused, so it is always perceptible when focused:

```svelte
<a
  href={`#libraries/edit/${lib.id}`}
  class="opacity-30 group-hover:opacity-100 group-focus-within:opacity-100 focus:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-400 rounded text-ink-500 hover:text-accent-400 transition-all p-0.5 flex-shrink-0"
  aria-label={`Library settings for ${lib.name}`}
  onclick={onClose}
>
  <Settings2 class="w-3.5 h-3.5" />
</a>
```

Key details:

| Class / attribute | Purpose |
|-------------------|---------|
| `opacity-30` | Resting state — slightly dimmed but never invisible; satisfies WCAG 2.4.7 |
| `group-hover:opacity-100` / `group-focus-within:opacity-100` | Fully reveals the link when the user hovers or tabs into the library row |
| `focus:opacity-100` | Ensures the link is fully visible when it holds focus directly |
| `focus-visible:ring-2 focus-visible:ring-accent-400` | Provides an explicit focus ring for keyboard navigation |
| `rounded` | Ensures the focus ring follows the element's shape |
| `aria-label={...lib.name...}` | Unique, descriptive label per library (WCAG 2.4.6) |

**Rule:** Never apply `opacity-0` to an element that can receive keyboard focus. Use `opacity-30` (or higher) as the minimum resting opacity so focus is always visible. If you add a new icon-only action link to the sidebar, follow the same resting-opacity pattern.

### ARIA landmarks

The app uses semantic HTML5 landmark elements so screen readers can navigate by region.

#### Authenticated app shell (`App.svelte`)

| Landmark | Element | Notes |
|----------|---------|-------|
| Main navigation | `<aside>` (inside `Sidebar.svelte`) | Desktop persistent sidebar |
| Primary content | `<main id="main-content">` | Target of the skip link |
| Mobile header | `<div>` + hamburger `<button>` | Not a landmark; sits above `<main>` only on small screens |

#### Login / signup page (`Auth.svelte`)

The unauthenticated page renders without the app shell, so it provides its own `<main>` landmark:

```svelte
<div class="min-h-screen …">
  <!-- Decorative background elements (no landmark role) -->
  <div class="absolute inset-0 overflow-hidden pointer-events-none">…</div>

  <main class="w-full max-w-md …">
    <!-- logo, heading, login/signup form -->
  </main>
</div>
```

| Landmark | Element | Notes |
|----------|---------|-------|
| Primary content | `<main>` (in `Auth.svelte`) | Wraps the entire login/signup card; ensures screen readers have a `<main>` landmark before authentication |

**Why this matters:** The WCAG [1.3.6 Identify Purpose](https://www.w3.org/WAI/WCAG21/Understanding/identify-purpose.html) guideline and best practice require every page to expose its primary content inside a `<main>` landmark so assistive-technology users can jump directly to it. Without `<main>`, the login page would be a flat, undifferentiated document from a screen reader's perspective.

> **Rule:** Every page — authenticated or not — must contain exactly one `<main>` landmark. For the authenticated shell, `<main id="main-content">` lives in `App.svelte`. For the pre-auth login page, `<main>` lives in `Auth.svelte`.

#### ARIA tab widget — Login/Sign Up toggle (`Auth.svelte`)

**WCAG criterion:** [4.1.2 Name, Role, Value](https://www.w3.org/WAI/WCAG21/Understanding/name-role-value.html) (Level A)

The Login/Sign Up toggle implements the [ARIA tab widget pattern](https://www.w3.org/WAI/ARIA/apg/patterns/tabs/) so that screen readers announce it as a labelled tab bar, not a pair of plain buttons.

```svelte
<!-- Tab bar: groups and labels the widget -->
<div
  role="tablist"
  aria-label="Authentication method"
  onkeydown={handleTabKeydown}
>
  <button
    id="login-tab"
    role="tab"
    aria-selected={isLogin}
    aria-controls="login-panel"
    tabindex={isLogin ? 0 : -1}
  >Login</button>

  <button
    id="signup-tab"
    role="tab"
    aria-selected={!isLogin}
    aria-controls="signup-panel"
    tabindex={!isLogin ? 0 : -1}
  >Sign Up</button>
</div>

<!-- Panels: one per tab (no tabindex — each panel contains focusable form elements) -->
<div id="login-panel"  role="tabpanel" aria-labelledby="login-tab"  hidden={!isLogin}>
  <!-- login form -->
</div>
<div id="signup-panel" role="tabpanel" aria-labelledby="signup-tab" hidden={isLogin}>
  <!-- sign-up form -->
</div>
```

**Key attributes:**

| Attribute | Element | Purpose |
|-----------|---------|---------|
| `role="tablist"` | Container `<div>` | Groups tab buttons into a single widget |
| `aria-label="Authentication method"` | Container `<div>` | Gives the tablist an accessible name announced by screen readers |
| `role="tab"` | Each `<button>` | Declares the button as a tab control |
| `aria-selected` | Each tab | `true` on the active tab; `false` on inactive tabs |
| `aria-controls` | Each tab | Associates the tab with its panel via the panel's `id` |
| `tabindex={isActive ? 0 : -1}` | Each tab | Implements the **roving tabindex** (see below) |
| `role="tabpanel"` | Each form panel | Declares the container as a tab panel |
| `aria-labelledby` | Each panel | Links the panel back to its controlling tab |
| `hidden` | Inactive panel | Hides the inactive panel from both display and the accessibility tree |

> **Note on `tabindex` for tabpanels:** These panels intentionally omit `tabindex="0"`. Each panel contains natively focusable elements (inputs and buttons), so adding `tabindex="0"` would create a redundant extra tab stop on the panel container before the first form control. Only panels with *no* focusable descendants should add `tabindex="0"` to remain reachable via keyboard. See [WAI-ARIA Authoring Practices — Tabs pattern](https://www.w3.org/WAI/ARIA/apg/patterns/tabs/).

**Roving tabindex:**
Only the active tab sits in the natural tab order (`tabindex="0"`); inactive tabs are removed from it (`tabindex="-1"`) but remain focusable programmatically. When the user tabs away from the active tab, focus moves to the first focusable element inside the active panel — inactive tabs are skipped, matching the expected [APG tab pattern](https://www.w3.org/WAI/ARIA/apg/patterns/tabs/) behaviour.

**Keyboard navigation (`handleTabKeydown`):**

| Key | Action |
|-----|--------|
| `ArrowRight` / `ArrowLeft` | Move focus to the next / previous tab and activate it |
| `Home` | Move focus and activation to the first tab (Login) |
| `End` | Move focus and activation to the last tab (Sign Up) |

**Why `hidden` instead of Svelte `{#if}`:**
The `hidden` HTML attribute is used on inactive panels rather than Svelte's `{#if}` block. Both panels stay in the DOM, so `aria-controls` references always point to a valid element. Removing a panel with `{#if}` would leave a dangling `aria-controls` reference and break the ARIA association.

### Page heading hierarchy

**WCAG criterion:** [1.3.1 Info and Relationships](https://www.w3.org/WAI/WCAG21/Understanding/info-and-relationships.html) (Level A) · [2.4.6 Headings and Labels](https://www.w3.org/WAI/WCAG21/Understanding/headings-and-labels.html) (Level AA)

Each page should contain exactly one visible `<h1>` when displaying primary content. In the authenticated app shell, the `<h1>` is owned by the active view component — not by any persistent shell element such as the sidebar.

```
App.svelte (shell)
├── Sidebar.svelte — brand name rendered as <p>, not <h1>
└── <main id="main-content">
    └── Dashboard.svelte (or Books.svelte, Libraries.svelte, …)
        └── <h1>Dashboard</h1>   ← the page's only <h1>
```

**Composite views:** Some view components delegate to sub-components that render their own heading hierarchy. For example, `Libraries.svelte` delegates to `LibraryView.svelte` (which contains the `<h1>`) or `LibraryForm.svelte` (which uses `<h2>` as the top heading within a card). Empty or transitional states (e.g., "Select a library") may omit the `<h1>` when there is no meaningful page title to display.

The sidebar brand name ("biblioteka") is styled to look prominent but uses a `<p>` element so the document outline contains exactly one `<h1>` per view:

```svelte
<!-- Sidebar.svelte — brand name -->
<p class="text-lg font-display font-bold tracking-tight">biblioteka</p>
```

Using `<h1>` here would create a second top-level heading on every authenticated page, corrupting the document outline seen by screen readers and automated accessibility tools.

**Heading levels in use:**

| Level | Where | Element / role |
|-------|-------|----------------|
| `h1` | Active view component (`Dashboard.svelte`, `Books.svelte`, etc.) | Native `<h1>` |
| `h2` | Sidebar navigation group labels | Native `<h2>` |
| `h2` and below | Content-area section headers | Native `<h2>`, `<h3>`, etc. |

**When adding a new page view**, the top-level heading for that page must be a native `<h1>`. Do not add an `<h1>` inside persistent shell elements (sidebar, header bar, footer).

### `aria-current` on active navigation links

**WCAG criterion:** [4.1.2 Name, Role, Value](https://www.w3.org/WAI/WCAG21/Understanding/name-role-value.html) (Level A)

Navigation links that represent the currently active view must carry `aria-current="page"`. Without this attribute, keyboard and screen-reader users have no programmatic way to determine which section is active — they can only infer it from visual styling, which is inaccessible.

#### Sidebar navigation (`Sidebar.svelte`)

Each top-level navigation link receives `aria-current` dynamically based on the `currentView` prop:

```svelte
<a
  href="#dashboard"
  aria-current={currentView === "dashboard" ? "page" : undefined}
  class="…"
  onclick={onClose}
>
  <LayoutDashboard class="w-5 h-5" />
  Dashboard
</a>
```

- Set `aria-current="page"` when the link represents the currently displayed view.
- Pass `undefined` (not `false`) for inactive links — `undefined` omits the attribute entirely, which is the correct behaviour. Using `aria-current="false"` is valid but adds noise and can confuse some assistive technologies.

#### Settings tab navigation (`Settings.svelte`)

The same pattern applies to settings sub-tabs, where the active tab is determined from the current `settingsSubPath`:

```svelte
<a
  href="#settings/account"
  aria-current={isActive ? "page" : undefined}
  class="…"
>
  Account
</a>
```

#### Rule for new navigation elements

Whenever you add a navigation link that points to a distinct view or sub-page, apply `aria-current={isActive ? "page" : undefined}`. Do **not** rely solely on CSS class changes to convey the active state.

### Labelled navigation groups (`Sidebar.svelte`)

**WCAG criterion:** [1.3.1 Info and Relationships](https://www.w3.org/WAI/WCAG21/Understanding/info-and-relationships.html) (Level A)

The navigation sidebar groups its links into named clusters (currently **Home** and **Libraries**). Without explicit semantics, these group labels are purely visual — assistive technologies see only a list of links and cannot determine which links belong to which group.

Each cluster is wrapped in a `role="group"` container labelled by a native `<h2>` heading. The heading is given a stable `id` so the group references it via `aria-labelledby`:

```svelte
<!-- Home group -->
<div role="group" aria-labelledby="sidebar-home-heading">
  <h2
    id="sidebar-home-heading"
    class="px-3 mb-2 text-[10px] font-semibold uppercase tracking-[0.15em] text-ink-500"
  >
    Home
  </h2>
  <div class="space-y-0.5">
    <a href="#dashboard" …>Dashboard</a>
    <a href="#books"     …>All Books</a>
  </div>
</div>

<!-- Libraries group -->
<div role="group" aria-labelledby="sidebar-libraries-heading">
  <div class="flex items-center justify-between px-3 mb-2">
    <h2
      id="sidebar-libraries-heading"
      class="text-[10px] font-semibold uppercase tracking-[0.15em] text-ink-500"
    >
      Libraries
    </h2>
    …
  </div>
  …library links…
</div>
```

**Why `role="group"` instead of `<nav>`?** The outer `<nav aria-label="Primary navigation">` already provides the landmark for the whole sidebar. Adding a second `<nav>` per cluster would inflate the number of navigation landmarks, making landmark-based navigation noisier. A `role="group"` groups the links semantically without creating extra landmarks.

**When adding a new navigation group:**

1. Wrap the cluster in `<div role="group" aria-labelledby="sidebar-<name>-heading">`.
2. Add an `<h2 id="sidebar-<name>-heading">` as the visual group label.
3. Add a test assertion in `Sidebar.test.ts` that `screen.getByRole("heading", { name: "<Name>", level: 2 })` is visible.

### Accessible labels for icon-only controls

**WCAG criterion:** [1.3.1 Info and Relationships](https://www.w3.org/WAI/WCAG21/Understanding/info-and-relationships.html) / [4.1.2 Name, Role, Value](https://www.w3.org/WAI/WCAG21/Understanding/name-role-value.html) (Level A)

Buttons and links that render only an icon (no visible text) and form inputs that cannot be paired with a visible `<label>` element — for example, inputs inside dynamically repeated rows — must have an explicit accessible name.

#### Icon-only links and buttons

Use `aria-label` on any `<a>` or `<button>` whose only child is an icon component. Add `aria-hidden="true"` on the icon element itself so screen readers announce only the control's label and do not also read out the SVG's internal title or path description:

```svelte
<!-- Sidebar: "Create library" — navigates to the new-library form -->
<a
  href="#libraries/new"
  aria-label="Create library"
  onclick={onClose}
>
  <Plus class="w-4 h-4" aria-hidden="true" />
</a>

<!-- Close button: renders only the X icon -->
<button
  onclick={navigateBack}
  aria-label="Close form"
>
  <X class="w-5 h-5" aria-hidden="true" />
</button>

<!-- Remove-folder button in a repeated list -->
<button
  type="button"
  onclick={() => { formPaths = formPaths.filter((_, idx) => idx !== i); }}
  aria-label="Remove folder"
  disabled={saving}
>
  <X class="w-4 h-4" aria-hidden="true" />
</button>
```

Without `aria-label`, screen readers announce these controls only by their SVG title or nothing at all, giving users no meaningful description of the action. Without `aria-hidden="true"` on the icon, some screen readers may announce both the control label **and** the SVG's internal title, causing a duplicate or confusing announcement.

#### Inputs in dynamic lists

When a form field is repeated (e.g., a list of folder paths), there may be no single `<label>` element that can be associated with every input via `for`/`id`. Use `aria-label` directly on the `<input>` with a distinguishing index when there are multiple items:

```svelte
<input
  type="text"
  aria-label={formPaths.length === 1 ? "Folder path" : `Folder path ${i + 1}`}
  bind:value={entry.value}
/>
```

When there is only one input in the list, omit the index to keep the label natural. When there are multiple inputs, append the 1-based position so screen reader users can distinguish them.

#### Checklist

- Every `<button>` or `<a>` that renders only an icon must have `aria-label` or `aria-labelledby`.
- Icon elements inside labeled controls must carry `aria-hidden="true"` to prevent duplicate announcements.
- Every `<input>` and `<select>` must have either a linked `<label for="...">` or an `aria-label` / `aria-labelledby`.
- `title` attributes are not a substitute for `aria-label`; they are advisory only and are not reliably announced.

### Maintaining accessibility

When editing the app shell or adding new persistent navigation elements:

1. Keep the skip link as the **first** child of the authenticated shell `<div>`.
2. If you add a new persistent region that users must bypass, add an additional skip link or update the existing one.
3. Every page — authenticated or not — must contain exactly one `<main>` landmark. For the authenticated shell this is `<main id="main-content">` in `App.svelte`; for the pre-auth login/signup page this is `<main>` in `Auth.svelte`. Do not remove or replace these elements with a generic `<div>`.
4. All interactive elements that are not natively focusable must have `tabindex="-1"` (receive focus programmatically only) or `tabindex="0"` (enter the natural tab order). Never use `tabindex` values greater than `0`.
5. Do not remove the `tabindex="-1"` attribute from `<main id="main-content">`. It is required by both the skip link (WCAG 2.4.1) and the SPA navigation focus effect (WCAG 2.4.3). Without it, `element.focus()` silently no-ops in most browsers.
6. Every icon-only link or button must have `aria-label`; the icon element inside the control must carry `aria-hidden="true"` to suppress redundant announcements; every unlabelled input must have `aria-label` or `aria-labelledby`. See [Accessible labels for icon-only controls](#accessible-labels-for-icon-only-controls) above.
7. Navigation links that represent the active view must carry `aria-current={isActive ? "page" : undefined}`. Tab-style buttons should use `aria-selected` instead (see item 9). See [`aria-current` on active navigation links](#aria-current-on-active-navigation-links) above.
8. Toggle switches (`<input type="checkbox">` styled as a switch) must carry `role="switch"` **and** an explicit `aria-checked` attribute, with an explicit `for`/`id` label association. See [`role="switch"` on toggle inputs](#roleswitch-on-toggle-inputs) below.
9. Tab-style navigation widgets (a set of buttons that show/hide panels) must use the ARIA tablist/tab/tabpanel pattern with roving tabindex and keyboard navigation (Arrow keys, Home, End). See [ARIA tab widget — Login/Sign Up toggle](#aria-tab-widget--loginsign-up-toggle-authsvelte) for the reference implementation.
10. Data tables must have `scope="col"` (or `scope="row"`) on every `<th>`. Visual-only columns (e.g., "Actions") must have an `sr-only` text label inside their `<th>`. State-toggle buttons in table rows must use action-oriented `aria-label` values. See [Table accessibility](#table-accessibility) below.
11. Sidebar navigation groups must use `role="group"` with `aria-labelledby` pointing to a native `<h2>` heading so screen readers announce the section name. See [Labelled navigation groups](#labelled-navigation-groups-sidebarsvelte) above.
12. Page view components should include a native `<h1>` for their primary content state. Composite views that delegate to sub-components (e.g., `Libraries.svelte` → `LibraryView.svelte`) may have the `<h1>` in the sub-component; empty or transitional states may omit it. Persistent shell elements (sidebar, header, footer) must never contain an `<h1>`. See [Page heading hierarchy](#page-heading-hierarchy) above.
13. Never apply `opacity-0` to an element that can receive keyboard focus. Use `opacity-30` (or higher) as the minimum resting opacity so the element is visible when focused. When the action is context-sensitive (e.g. per-library settings links), include the context in the `aria-label` so each link has a unique, descriptive name. See [Focus visible — Library settings link](#focus-visible--library-settings-link-sidebarsvelte) above.
14. Run `pnpm run check` — `svelte-check` will surface missing `alt` attributes and other common issues.

### Form accessibility

All form inputs must be programmatically associated with a visible label or carry a descriptive `aria-label`. `LibraryForm.svelte` serves as the canonical reference for these patterns.

#### Explicit `<label for>` / `id` pairing

Use an explicit `for`/`id` association for text inputs that have a visible label. This is the strongest and most broadly supported technique.

```svelte
<label for="lib-name" class="block text-sm font-medium …">Name</label>
<input id="lib-name" type="text" bind:value={formName} … />
```

Screen readers will announce the label text whenever the input receives focus.

#### Dynamic `aria-label` for repeated inputs

When a form contains a variable-length list of inputs of the same type (e.g. multiple folder-path fields), use a dynamic `aria-label` that includes the item's position so screen-reader users can distinguish them.

```svelte
{#each formPaths as entry, i}
  <input
    aria-label={formPaths.length === 1 ? "Folder path" : `Folder path ${i + 1}`}
    …
  />
{/each}
```

- Use the singular label (e.g. `"Folder path"`) when there is only one item — the ordinal is unnecessary and adds noise.
- Use a 1-based counter (e.g. `"Folder path 1"`, `"Folder path 2"`) when there are multiple items.

#### `aria-label` on icon-only controls

Links and buttons whose visible content is solely an icon (SVG) must carry an `aria-label` so assistive technologies announce an intelligible action name. The icon itself must also carry `aria-hidden="true"` to prevent screen readers from reading both the control label and the SVG's internal title.

```svelte
<!-- Close / cancel button -->
<button aria-label="Close form" onclick={navigateBack}>
  <X class="w-5 h-5" aria-hidden="true" />
</button>

<!-- Remove item button inside a list -->
<button aria-label="Remove folder" …>
  <X class="w-4 h-4" aria-hidden="true" />
</button>
```

When you add a new icon-only link or button, always supply both `aria-label` on the element and `aria-hidden="true"` on the icon component. `svelte-check` does **not** automatically detect missing labels or missing `aria-hidden`, so this must be reviewed manually.

#### `autocomplete` on credential inputs

**WCAG criterion:** [1.3.5 Identify Input Purpose](https://www.w3.org/WAI/WCAG21/Understanding/identify-input-purpose.html) (Level AA)

Password inputs must carry a valid `autocomplete` token so that password managers and autofill implementations can correctly identify the field's purpose. Without `autocomplete`, browsers may misclassify the fields, offer to save them as plain text, or fail to auto-fill them — degrading both usability and security.

`settings/AccountTab.svelte` uses three password fields with the following tokens:

```svelte
<!-- Current/existing password -->
<input type="password" autocomplete="current-password" … />

<!-- New password (set or confirm) -->
<input type="password" autocomplete="new-password" … />
<input type="password" autocomplete="new-password" … />
```

| Token | When to use |
|-------|-------------|
| `current-password` | The user's existing credential (used for verification before allowing a change) |
| `new-password` | A newly chosen password the user is setting or confirming |

**Do not** leave `type="password"` inputs without `autocomplete`. Browsers may still infer the purpose, but the explicit attribute is required by WCAG 1.3.5 and ensures reliable cross-browser behaviour.

#### `fieldset` and `legend` for grouped inputs

**WCAG criterion:** [1.3.1 Info and Relationships](https://www.w3.org/WAI/WCAG21/Understanding/info-and-relationships.html) (Level A)

When a form contains a group of related inputs that share a common heading (e.g., a set of folder paths that all belong to "Folders"), wrap them in a `<fieldset>` with a `<legend>` instead of a plain `<div>` with a visual label. Screen readers announce the group's legend when focus enters any field inside it, giving users the context they need to understand the relationship.

```svelte
<fieldset class="border-none p-0 m-0">
  <legend class="block text-sm font-medium …">
    Folders <span aria-hidden="true">*</span>
  </legend>
  <div class="space-y-2">
    {#each formPaths as entry, i (entry.id)}
      <input
        type="text"
        aria-label={formPaths.length === 1 ? "Folder path" : `Folder path ${i + 1}`}
        …
      />
    {/each}
  </div>
</fieldset>
```

- The `<legend>` text becomes the accessible name for the entire group; all inputs inside are announced in context of it.
- Use `class="border-none p-0 m-0"` (or equivalent) to remove the browser's default `<fieldset>` border and padding while preserving the semantic grouping.
- The required-field asterisk (`*`) should carry `aria-hidden="true"` on its containing `<span>` so screen readers do not announce "asterisk" mid-sentence — mark required fields with `aria-required="true"` on the `<input>` instead.

#### `aria-describedby` for inline validation errors

**WCAG criterion:** [1.3.1 Info and Relationships](https://www.w3.org/WAI/WCAG21/Understanding/info-and-relationships.html) / [3.3.1 Error Identification](https://www.w3.org/WAI/WCAG21/Understanding/error-identification.html) (Level A)

When an input has an associated error message that appears inline below it, use `aria-describedby` to link the input to the error element. Without this link, screen reader users hear the input's label but not the error — they must navigate to the error paragraph separately.

```svelte
<input
  id="lib-name"
  type="text"
  bind:value={formName}
  aria-required="true"
  aria-invalid={nameError ? true : undefined}
  aria-describedby={nameError ? "lib-name-error" : undefined}
/>
{#if nameError}
  <p id="lib-name-error" role="alert" class="text-sm text-danger-600 …">
    {nameError}
  </p>
{/if}
```

**Key points:**

| Attribute | Element | When to set |
|-----------|---------|-------------|
| `aria-invalid="true"` | `<input>` | When the field has a validation error |
| `aria-describedby="<error-id>"` | `<input>` | When the error element is visible |
| `id="<error-id>"` | Error `<p>` | Always — must match `aria-describedby` |
| `role="alert"` | Error `<p>` | Makes the error message announce immediately when it appears |

- Set `aria-invalid` and `aria-describedby` conditionally — only when the error is present. Passing `undefined` removes the attribute; passing `false` for `aria-invalid` is valid but adds noise and may confuse some screen readers.
- For grouped inputs (e.g., folder paths sharing a single error), point every input's `aria-describedby` to the shared error element's `id`.

#### `role="switch"` on toggle inputs

**WCAG criterion:** [4.1.2 Name, Role, Value](https://www.w3.org/WAI/WCAG21/Understanding/name-role-value.html) (Level A)

A visually styled toggle switch implemented with `<input type="checkbox">` does not automatically communicate its "switch" semantics to assistive technologies. Screen readers announce it as a plain checkbox, which is misleading when the control represents a binary on/off state rather than a multi-select option.

Add `role="switch"` **and** `aria-checked` to any `<input type="checkbox">` that renders as a toggle. `role="switch"` alone is insufficient — some assistive technologies cannot reliably infer the current state from the native `checked` property when the element has an overriding ARIA role; `aria-checked` makes the state explicit.

Use an explicit `for`/`id` association between the `<label>` and the input alongside `role="switch"` (supplementing any implicit wrapper-label approach) to maximize cross-browser screen reader compatibility:

```svelte
<label for="lib-monitored" class="…">Monitor for new content</label>
<input
  id="lib-monitored"
  type="checkbox"
  role="switch"
  aria-checked={formMonitored}
  bind:checked={formMonitored}
  class="sr-only peer"
/>
```

In Svelte 5 (runes mode), binding a boolean to `aria-checked` is correct: Svelte serializes `aria-*` attributes to their string representation (`"true"` / `"false"`) rather than removing them, so `aria-checked={false}` renders as `aria-checked="false"` as required by the ARIA spec.

With `role="switch"` and `aria-checked`, assistive technologies announce the control as a _switch_ and report its state as _on_ or _off_ (rather than _checked_ or _unchecked_), which is the semantically correct announcement for a toggle.

**Why `aria-checked` must be set explicitly**

`bind:checked` keeps the underlying `checked` DOM property in sync, but some screen reader and browser combinations do not reliably derive the `aria-checked` value from the DOM property when `role="switch"` is present. Setting `aria-checked` explicitly as an attribute ensures the state is always exposed correctly to the accessibility tree.

**Guidelines:**

- Apply `role="switch"` and `aria-checked={booleanState}` whenever a `<input type="checkbox">` is styled to look like a toggle switch.
- Always set `aria-checked` explicitly as an attribute bound to the same reactive variable as `bind:checked` (e.g. `aria-checked={myState}`). Do **not** rely solely on `bind:checked` to expose state, as some browser/screen reader combinations do not derive `aria-checked` from the DOM `checked` property when `role="switch"` is present.
- Use an explicit `<label for="…">` / `id="…"` association in addition to any implicit wrapper label.
- The control must still have an accessible name — either via a paired `<label>` element (preferred) or `aria-label`.
- Do **not** use `role="switch"` on checkboxes that genuinely represent a tri-state or multi-select option; use plain `type="checkbox"` there.

`LibraryForm.svelte` is the canonical reference implementation.

#### Checklist for new forms

When adding or editing a form component:

1. Every `<input>`, `<select>`, and `<textarea>` has either a `<label for="…">` or an `aria-label`.
2. `<label for>` values match the corresponding `id` exactly — a mismatch silently breaks the association.
3. Related inputs that share a common heading are wrapped in a `<fieldset>` with a `<legend>`. See [`fieldset` and `legend` for grouped inputs](#fieldset-and-legend-for-grouped-inputs) above.
4. Icon-only buttons (`<button>` with SVG content and no text) carry an `aria-label`.
5. Repeated inputs in `{#each}` blocks use a dynamic, positionally-distinct `aria-label`.
6. Inputs with inline validation errors set `aria-invalid="true"` and `aria-describedby="<error-id>"` when the error is visible. See [`aria-describedby` for inline validation errors](#aria-describedby-for-inline-validation-errors) above.
7. Password inputs (`type="password"`) carry `autocomplete="current-password"` or `autocomplete="new-password"` as appropriate.
8. Toggle switches (`<input type="checkbox">` styled as a switch) carry both `role="switch"` and `aria-checked={booleanState}`. Use an explicit `for`/`id` label association. See [`role="switch"` on toggle inputs](#roleswitch-on-toggle-inputs) above.
9. Tab-style widgets that show/hide panels use the ARIA tablist/tab/tabpanel pattern with roving tabindex, `aria-selected`, `aria-controls`/`aria-labelledby`, and keyboard navigation (Arrow keys, Home, End). See [ARIA tab widget — Login/Sign Up toggle](#aria-tab-widget--loginsign-up-toggle-authsvelte) for the reference implementation.
10. Run `pnpm run check` after your changes — `svelte-check` will catch missing `alt` on images and some label issues.

### Inline confirmation dialogs for destructive actions

**WCAG criterion:** [4.1.2 Name, Role, Value](https://www.w3.org/WAI/WCAG21/Understanding/name-role-value.html) / [2.1.1 Keyboard](https://www.w3.org/WAI/WCAG21/Understanding/keyboard.html) / [3.3.4 Error Prevention](https://www.w3.org/WAI/WCAG21/Understanding/error-prevention-legal-financial-data.html) (Level A / AA)

Destructive actions such as deleting an API key or Kobo sync token must not use `window.confirm()`. The browser's native confirm dialog has no ARIA role, no focus management, is not styleable, and produces inconsistent screen reader behaviour across browsers. Instead, render an inline `role="alertdialog"` directly in the component.

#### Pattern

Use a piece of reactive state (`pendingDeleteKey`, `pendingDeleteToken`, etc.) to track which item is pending deletion. When the user clicks the initial delete trigger, set that state; the row or card swaps its delete button for an inline confirmation:

```svelte
<script lang="ts">
  import { tick } from "svelte";
  import { autofocusFirstButton } from "../../lib/actions";

  let pendingDeleteItem: { id: string; name: string } | null = $state(null);

  function handleDeleteItem(id: string, name: string) {
    pendingDeleteItem = { id, name };
  }

  async function confirmDeleteItem() {
    if (!pendingDeleteItem) return;
    const { id } = pendingDeleteItem;
    pendingDeleteItem = null;
    await deleteItem(id); // your API call
    // filter item from list…
  }

  async function cancelDeleteItem() {
    const id = pendingDeleteItem?.id; // capture id BEFORE clearing state
    pendingDeleteItem = null;
    await tick();
    // Return focus to the trigger button after cancellation
    document.querySelector<HTMLElement>(`[data-delete-trigger="${id}"]`)?.focus();
  }
</script>

{#if pendingDeleteItem?.id === item.id}
  <div
    role="alertdialog"
    aria-modal="false"
    aria-labelledby={`confirm-label-${item.id}`}
    tabindex="-1"
    use:autofocusFirstButton
    onkeydown={(e: KeyboardEvent) => { if (e.key === "Escape") cancelDeleteItem(); }}
  >
    <span id={`confirm-label-${item.id}`}>Delete "{item.name}"?</span>
    <button onclick={confirmDeleteItem}>Delete</button>
    <button onclick={cancelDeleteItem}>Cancel</button>
  </div>
{:else}
  <button
    data-delete-trigger={item.id}
    onclick={() => handleDeleteItem(item.id, item.name)}
    aria-label={`Delete ${item.name}`}
  >Delete</button>
{/if}
```

**Preferred approach:** Use the [`DeleteConfirmation.svelte`](#deleteconfirmationsvelte) UI component — it encapsulates the `role="alertdialog"`, autofocus, and Escape-to-dismiss boilerplate so you only need to supply `itemId`, `itemName`, `onConfirm`, and `onCancel`. `LibraryForm.svelte`, `APIKeysTab.svelte`, and `KoboTab.svelte` are the canonical reference implementations.

If you cannot use `DeleteConfirmation` (e.g., the confirmation UI needs a non-standard layout), implement the inline pattern manually using the code skeleton above as a starting point.

#### `autofocusFirstButton` Svelte action (`lib/actions.ts`)

`autofocusFirstButton` is a Svelte action that moves keyboard focus to the first `<button>` inside a container after the next microtask, ensuring child elements have rendered before focus is requested:

```ts
import { autofocusFirstButton } from "../../lib/actions"; // adjust path to match your component's depth

// In a template:
// <div use:autofocusFirstButton>…</div>
```

Apply `use:autofocusFirstButton` to the confirmation dialog container so focus is moved into it automatically when it mounts. This satisfies the keyboard-focus management requirement for modal and inline dialog patterns.

**Note:** `autofocusFirstButton` focuses the first `<button>` in DOM order. If you want focus to land on the safer Cancel option, place Cancel before Delete in the DOM. The current implementations focus the Delete button first; this is a known P2 issue.

#### Guidelines

- Never use `window.confirm()` for destructive confirmations.
- Use `role="alertdialog"` (not `role="dialog"`) for inline confirmations — `alertdialog` causes screen readers to announce its content immediately.
- Always capture the pending item's `id` into a local variable **before** clearing the pending state in cancel handlers; clearing state first causes `querySelector` to search for `[data-delete-trigger="undefined"]` and silently drops focus.
- Add `use:autofocusFirstButton` to the dialog container so keyboard focus enters the dialog when it opens.
- Add an `onkeydown` Escape handler to dismiss the dialog; call the cancel function.
- Add `data-delete-trigger` on the original trigger button so focus can be restored on cancel (WCAG 2.4.3 Focus Order).
- Keep only one confirmation dialog open at a time — the pending state should be a single nullable value, not an array.

### Table accessibility

**WCAG criterion:** [1.3.1 Info and Relationships](https://www.w3.org/WAI/WCAG21/Understanding/info-and-relationships.html) (Level A)

Data tables must programmatically associate each header cell with the data cells it describes. Without this association, screen readers announce cell contents as isolated values with no column or row context.

#### `scope` attribute on column headers

Add `scope="col"` to every `<th>` that acts as a column header:

```svelte
<thead>
  <tr>
    <th scope="col">Title</th>
    <th scope="col">Publisher</th>
    <th scope="col">Language</th>
    <th scope="col">Pages</th>
  </tr>
</thead>
```

When a `<th>` spans rows instead of columns (a row header), use `scope="row"`. For most flat, non-hierarchical tables in Biblioteka, `scope="col"` is sufficient.

`UsersTab.svelte` is the canonical reference implementation: its "Name", "Email", "Type", "Role", and "Joined" headers all carry `scope="col"`.

#### Accessible label for visual-only header cells

An "Actions" column typically has no visible heading — its purpose is implied by the buttons in each row. A blank `<th>` is still announced by screen readers (often as an empty cell), which can be confusing. Use an `sr-only` span to provide a descriptive label that sighted users cannot see:

```svelte
<th scope="col">
  <span class="sr-only">Actions</span>
</th>
```

- Do **not** leave a `<th>` completely empty. An empty header cell gives screen-reader users no context for the column.
- `sr-only` is a Tailwind utility (equivalent to `position: absolute; width: 1px; height: 1px; overflow: hidden; …`) that hides the text visually while keeping it in the accessibility tree.

#### Checklist for new tables

When adding a data table component:

1. Every column `<th>` has `scope="col"`.
2. Columns whose purpose is visually implied (e.g., "Actions") have `<span class="sr-only">Actions</span>` inside their `<th>`.
3. If a `<th>` spans rows, it has `scope="row"`.
4. Do not use `<td>` for header cells — use `<th scope="…">` so the relationship is semantically clear.
5. Inline state-toggle buttons (whose visible text reflects the current state) must carry an action-oriented `aria-label` so screen-reader users hear what the button will *do*, not just what the current state *is*. See [Action-oriented labels for state-toggle buttons](#action-oriented-labels-for-state-toggle-buttons) below.

#### Action-oriented labels for state-toggle buttons

**WCAG criterion:** [4.1.2 Name, Role, Value](https://www.w3.org/WAI/WCAG21/Understanding/name-role-value.html) (Level A)

When a table row contains a button whose visible text describes the *current state* of the row (e.g. "Admin" / "User"), screen-reader users hear only the state — they cannot tell what action the button will perform. Use `aria-label` to give the button an *action-oriented* accessible name that names both the operation and its target.

```svelte
<button
  onclick={() => toggleAdmin(u)}
  aria-label={u.is_admin
    ? `Remove admin role from ${u.name || u.email}`
    : `Grant admin role to ${u.name || u.email}`}
>
  {u.is_admin ? "Admin" : "User"}
</button>
```

Key points:

- The visible button text keeps its **state-description** role for sighted users.
- The `aria-label` overrides the accessible name with the **action** the button will perform and the **target** it will affect (by name, falling back to email if the display name is absent).
- Use `u.name || u.email` (not just `u.name`) so the label remains meaningful for accounts that have no display name.
- After the toggle completes, Svelte reactively recomputes `u.is_admin`, so the `aria-label` automatically switches to the opposite operation on the next render — no manual state management is required.

`UsersTab.svelte` is the canonical reference implementation: its role-toggle button uses exactly this pattern.

**Do not** use the current-state text alone as the accessible name:

```svelte
<!-- ✗ Announces "Admin" — tells the user the state, not the action -->
<button onclick={() => toggleAdmin(u)}>
  {u.is_admin ? "Admin" : "User"}
</button>
```

**Do** add an action-oriented `aria-label`:

```svelte
<!-- ✓ Announces "Remove admin role from Alice" — tells the user what will happen -->
<button
  onclick={() => toggleAdmin(u)}
  aria-label={u.is_admin
    ? `Remove admin role from ${u.name || u.email}`
    : `Grant admin role to ${u.name || u.email}`}
>
  {u.is_admin ? "Admin" : "User"}
</button>
```

### Accessibility tests

Accessibility regressions are locked in by dedicated test files. Keep all of these tests green. If you restructure a component, update its test to match.

#### `App.test.ts`

`frontend/src/App.test.ts` verifies the authenticated shell:

- `document.title` is set from `routerStore.pageTitle` on mount (WCAG 2.4.2).
- The skip link exists and has the label _"Skip to main content"_.
- The `<main>` landmark has `id="main-content"`.
- The skip link appears **before** the sidebar in DOM order.
- Clicking the skip link moves keyboard focus to `<main>`.
- Keyboard focus is **not** moved to `<main>` on initial mount — the `focusEffectMounted` guard prevents the focus effect from running on first render so a hard refresh does not steal focus from the browser UI (WCAG 2.4.3).

#### `Auth.test.ts`

`frontend/src/components/Auth.test.ts` verifies the ARIA tab widget on the login/signup page (WCAG 4.1.2). It uses `@testing-library/user-event` to simulate real browser interactions and `await tick()` to flush Svelte 5 reactivity before asserting. Three tests are included:

1. **`has a main landmark region`** — asserts the login page exposes a `<main>` landmark (WCAG 1.3.6).
2. **`renders tab buttons with correct ARIA attributes`** — asserts the `role="tablist"` container is present and that each tab carries the correct `aria-selected` and `aria-controls` attributes on initial render.
3. **`renders tab panels with correct ARIA attributes`** — asserts both `role="tabpanel"` elements (including the hidden one) carry the correct `aria-labelledby` back-reference.
4. **`switches ARIA state when the Sign Up tab is clicked`** — simulates a user click on the Sign Up tab and verifies that `aria-selected` values update reactively and the Sign Up panel loses its `hidden` attribute.

> **Testing note:** Each test wraps `render(Auth)` in an `async renderAuth()` helper that calls `await tick()` after mounting. This is necessary because Svelte 5 defers reactive updates; without the tick the DOM may not reflect the initial `$state` values when the first `expect` runs. The `afterEach(cleanup)` guard ensures the JSDOM is cleared between tests to prevent state bleed.

#### `Sidebar.test.ts`

`frontend/src/components/Sidebar.test.ts` verifies that the navigation sidebar uses semantic `<a>` anchor links (not `<button>` elements) for in-app navigation, that `aria-current` state is applied correctly (WCAG 4.1.2), and that library-settings links meet accessibility requirements (WCAG 2.4.6, 2.4.7). Eight tests are included:

1. **`renders Dashboard, All Books, and Settings as links with correct hrefs`** — asserts that the three primary nav items are rendered as `role="link"` elements with the correct hash `href` values (`#dashboard`, `#books`, `#settings`).
2. **`renders Logout as a button, not a link`** — asserts that Logout is a `role="button"` element, which is semantically correct because it triggers an action (sign-out) rather than navigating to a URL.
3. **`sets aria-current='page' on the active navigation link`** — renders with `currentView="settings"` and asserts that the Settings link carries `aria-current="page"` while Dashboard does not.
4. **`sets aria-current='page' on the active library link`** — renders with `currentView="libraries"` and `subPath="1"` and asserts that only the matching library entry link receives `aria-current="page"`.
5. **`library settings links include the library name in aria-label (WCAG 2.4.6)`** — renders with two mock libraries and asserts each Library-settings link has a unique `aria-label` that includes the library name (e.g., "Library settings for Fiction") so screen readers can distinguish between them (WCAG 2.4.6 Headings and Labels).
6. **`library settings links are not fully transparent by default (WCAG 2.4.7)`** — asserts that the Library-settings link does not carry `opacity-0` and instead carries `opacity-30`, ensuring the element is perceivable when focused by keyboard (WCAG 2.4.7 Focus Visible).
7. **`renders navigation group labels as headings`** — renders with `currentView="dashboard"` and asserts that the "Home" and "Libraries" group labels are exposed as `role="heading"` elements at level 2 (WCAG 1.3.1).
8. **`does not render the app name as a heading`** — asserts that the brand name ("Biblioteka") is rendered as a `<p>` element, not an `<h1>`, so it does not create a duplicate top-level heading (WCAG 1.3.1).

> **Mocking note:** The test file mocks `authStore`, `libraryStore`, `api.getVersion`, and all `lucide-svelte` icon components. The icon mocks are necessary because Lucide icons are ESM-only packages that cannot render in JSDOM; replacing them with no-ops keeps the test focused on DOM structure. `afterEach(cleanup)` prevents DOM leakage between tests.

#### `LibraryForm.test.ts`

`frontend/src/components/libraries/LibraryForm.test.ts` verifies the accessibility attributes on the library create/edit form (WCAG 1.3.1, 3.3.1, 4.1.2). Tests are organised in three `describe` blocks.

**"LibraryForm accessibility" — six tests:**

1. **`marks the name input as aria-required`** — asserts the `#lib-name` text input carries `aria-required="true"`.
2. **`marks folder path inputs as aria-required`** — asserts the folder path input carries `aria-required="true"`.
3. **`shows required indicator (*) on Name and Folders labels`** — asserts both the `<label for="lib-name">` and the `<fieldset>` legend contain a `<span aria-hidden="true">` with text `*`, so the required indicator is visible but not read aloud by screen readers (the `aria-required` attribute serves as the machine-readable signal instead).
4. **`shows inline name error with aria-invalid when submitting empty name`** — submits the form with no name, then asserts the name input is marked `aria-invalid="true"`, references the error via `aria-describedby="lib-name-error"`, and that the error element carries `role="alert"` so it is announced immediately by assistive technologies.
5. **`shows inline folder error with aria-invalid when submitting empty paths`** — fills in the name (to pass name validation) then submits with no folder paths; asserts the folder input becomes `aria-invalid="true"` with `aria-describedby="lib-folders-error"` and the error element carries `role="alert"`.
6. **`does not show aria-invalid or error messages before submission`** — asserts that neither `aria-invalid` nor the error message elements are present on initial render, preventing premature error announcements.
7. **`associates the switch input with its label via for/id`** — asserts that the monitor toggle input has `id="lib-monitored"` and that a `<label>` element with `for="lib-monitored"` is present, verifying the explicit label association required by WCAG 1.3.1.
8. **`has aria-checked reflecting the unchecked state by default`** — asserts the monitor toggle carries `aria-checked="false"` on initial render (formMonitored defaults to `false`).
9. **`updates aria-checked when toggled`** — simulates a click on the monitor toggle and asserts `aria-checked` updates to `"true"`, verifying that the attribute stays in sync with the bound state.
10. **`renders the organization type dropdown`** — asserts a `<select>` element for the file organization setting is present in the form.
11. **`has three options with correct values`** — asserts the organization dropdown contains exactly the three options `book_per_folder`, `book_per_file`, and `none`.
12. **`defaults to book_per_folder in create mode`** — asserts the organization dropdown defaults to `book_per_folder` when the form is used to create a new library.
13. **`has a label with text File Organization`** — asserts the organization dropdown is paired with a visible label containing the text "File Organization".

**"LibraryForm monitor toggle switch" — three tests:**

1. **`associates the switch input with its label via for/id`** — asserts the `#lib-monitored` checkbox is linked to its visible label by the `for`/`id` pairing and carries `role="switch"`.
2. **`has aria-checked reflecting the unchecked state by default`** — asserts the unchecked toggle exposes `aria-checked="false"` as a DOM attribute (not just the `checked` property).
3. **`updates aria-checked when toggled`** — fires a click on the input; asserts both `input.checked` becomes `true` and `aria-checked` updates to `"true"`.

**"LibraryForm organization type dropdown" — four tests:** verify that the file-organization `<select>` renders with the correct options, defaults to `book_per_folder` in create mode, and is associated with a visible label.

> **Testing note:** Each test calls `await tick()` after `render()` to flush Svelte 5 reactive state before asserting. `afterEach(cleanup)` removes the rendered component from JSDOM between tests.

#### `APIKeysTab.test.ts`

`frontend/src/components/settings/APIKeysTab.test.ts` verifies the inline destructive confirmation dialog in the API keys settings tab (WCAG 2.1.1, 3.3.4, 4.1.2). Seven tests in the **"APIKeysTab delete confirmation"** `describe` block:

1. **`does not call deleteAPIKey when Delete button is clicked (shows confirmation instead)`** — clicks the delete trigger for an API key and asserts `deleteAPIKey` is **not** called, confirming that no deletion occurs until the user confirms.
2. **`shows inline confirmation dialog when Delete is clicked`** — clicks the delete trigger and asserts the `role="alertdialog"` confirmation panel appears in the DOM.
3. **`dismisses confirmation dialog when Cancel is clicked`** — opens the confirmation, then clicks Cancel; asserts the dialog is removed from the DOM.
4. **`calls deleteAPIKey after confirming deletion`** — opens the confirmation, clicks the confirm Delete button, and asserts `deleteAPIKey` is called with the correct key ID.
5. **`only shows confirmation for the clicked key, not all keys`** — seeds two API keys, clicks delete on the first, and asserts the dialog appears only for that key while the second key row remains unaffected.
6. **`dismisses confirmation dialog when Escape is pressed`** — opens the confirmation dialog, dispatches a `keydown` event with `key="Escape"`, and asserts the dialog is removed.
7. **`moves focus to the Delete confirm button when dialog opens`** — opens the confirmation dialog and asserts the Delete button inside the `alertdialog` receives focus, verifying the `autofocusFirstButton` action is working.

> **Mocking note:** The test file mocks `api.listAPIKeys` (returns two pre-seeded keys), `api.createAPIKey`, `api.deleteAPIKey` (resolves immediately), `clipboard.copyToClipboard`, and all `lucide-svelte` icon components. `afterEach` calls `cleanup()` and `vi.clearAllMocks()` to prevent state leakage between tests.

#### `KoboTab.test.ts`

`frontend/src/components/settings/KoboTab.test.ts` verifies the inline destructive confirmation dialog in the Kobo token settings tab (WCAG 2.1.1, 3.3.4, 4.1.2). Seven tests in the **"KoboTab delete confirmation"** `describe` block, mirroring the structure of `APIKeysTab.test.ts`:

1. **`does not call deleteKoboToken when Delete button is clicked (shows confirmation instead)`** — asserts deletion does not fire on the first click.
2. **`shows inline confirmation dialog when Delete is clicked`** — asserts the `role="alertdialog"` panel appears.
3. **`dismisses confirmation dialog when Cancel is clicked`** — asserts Cancel hides the dialog.
4. **`calls deleteKoboToken after confirming deletion`** — asserts `deleteKoboToken` is called with the correct token ID after confirmation.
5. **`only shows confirmation for the clicked token, not all tokens`** — asserts only the target token's row shows the dialog.
6. **`dismisses confirmation dialog when Escape is pressed`** — asserts the Escape key closes the dialog.
7. **`moves focus to the Delete confirm button when dialog opens`** — asserts the Delete button inside the `alertdialog` receives focus.

> **Mocking note:** The test file mocks `api.listKoboTokens`, `api.createKoboToken`, `api.deleteKoboToken`, `clipboard.copyToClipboard`, and all `lucide-svelte` icon components. `afterEach` calls `cleanup()` and `vi.clearAllMocks()`.

#### `UsersTab.test.ts`

`frontend/src/components/settings/UsersTab.test.ts` verifies that the Users table in the admin settings panel meets accessibility requirements (WCAG 1.3.1, 4.1.2). Two tests are included:

1. **`marks each table header as a column header`** — seeds the component with one cached user to bypass the initial loading path, then asserts that each of the five `<th>` elements — **Name**, **Email**, **Type**, **Role**, and **Joined** — is exposed as a `role="columnheader"` and carries `scope="col"`.

2. **`gives toggle-admin buttons descriptive accessible names`** — seeds the component with three users (one matching the logged-in admin, one regular user, one additional admin), then asserts that the two role-toggle buttons carry action-oriented `aria-label` values: `"Grant admin role to Reader User"` for the non-admin entry and `"Remove admin role from Staff User"` for the admin entry. The logged-in user's row renders a non-interactive badge instead of a button, so it is excluded from these assertions.

> **Mocking note:** The test file mocks `authStore` (current admin user), `api.listUsers` (returns a resolved empty array to prevent uncaught-promise warnings), and all `lucide-svelte` icon components. `cachedUsers` is passed as a prop to seed the rendered table immediately, avoiding the need for async load completion. `afterEach(cleanup)` prevents DOM leakage between tests.

#### `APIKeysTab.test.ts`

`frontend/src/components/settings/APIKeysTab.test.ts` verifies the inline confirmation dialog pattern for API key deletion (WCAG 2.1.1, 4.1.2). Seven tests are included in the `APIKeysTab delete confirmation` describe block:

1. **`does not call deleteAPIKey when Delete button is clicked (shows confirmation instead)`** — clicks the delete trigger for an API key and asserts `deleteAPIKey` is **not** called, confirming that no deletion occurs until the user confirms.
2. **`shows inline confirmation dialog when Delete is clicked`** — clicks the delete trigger and asserts the `role="alertdialog"` confirmation panel appears in the DOM.
3. **`dismisses confirmation dialog when Cancel is clicked`** — opens the confirmation, then clicks Cancel; asserts the dialog is removed from the DOM.
4. **`calls deleteAPIKey after confirming deletion`** — opens the confirmation, clicks the confirm Delete button, and asserts `deleteAPIKey` is called with the correct key ID.
5. **`only shows confirmation for the clicked key, not all keys`** — seeds two API keys, clicks delete on the first, and asserts the dialog appears only for that key while the second key row remains unaffected.
6. **`dismisses confirmation dialog when Escape is pressed`** — opens the confirmation dialog, dispatches a `keydown` event with `key="Escape"`, and asserts the dialog is removed.
7. **`moves focus to the Delete confirm button when dialog opens`** — opens the confirmation dialog and asserts the Delete button inside the `alertdialog` receives focus, verifying the `autofocusFirstButton` action is working.

> **Mocking note:** The test file mocks `api.listAPIKeys` (returns two pre-seeded keys), `api.createAPIKey`, `api.deleteAPIKey` (resolves immediately), `clipboard.copyToClipboard`, and all `lucide-svelte` icon components. `afterEach` calls `cleanup()` and `vi.clearAllMocks()` to prevent state leakage between tests.

#### `KoboTab.test.ts`

`frontend/src/components/settings/KoboTab.test.ts` verifies the inline confirmation dialog pattern for Kobo sync token deletion (WCAG 2.1.1, 4.1.2). Seven tests are included in the `KoboTab delete confirmation` describe block, mirroring the structure of `APIKeysTab.test.ts`:

1. **`does not call deleteKoboToken when Delete button is clicked (shows confirmation instead)`** — asserts deletion does not fire on the first click.
2. **`shows inline confirmation dialog when Delete is clicked`** — asserts the `role="alertdialog"` panel appears.
3. **`dismisses confirmation dialog when Cancel is clicked`** — asserts Cancel hides the dialog.
4. **`calls deleteKoboToken after confirming deletion`** — asserts `deleteKoboToken` is called with the correct token ID after confirmation.
5. **`only shows confirmation for the clicked token, not all tokens`** — asserts only the target token's row shows the dialog.
6. **`dismisses confirmation dialog when Escape is pressed`** — asserts the Escape key closes the dialog.
7. **`moves focus to the Delete confirm button when dialog opens`** — asserts the Delete button inside the `alertdialog` receives focus.

> **Mocking note:** The test file mocks `api.listKoboTokens`, `api.createKoboToken`, `api.deleteKoboToken`, `clipboard.copyToClipboard`, and all `lucide-svelte` icon components. `afterEach` calls `cleanup()` and `vi.clearAllMocks()`.

---

## Unit tests

The following test suites cover reactive stores and the API client. Unlike the accessibility tests above, these tests verify logic and state management rather than DOM structure.

### `router.test.ts`

`frontend/src/stores/router.test.ts` exercises `routerStore`, the hash-based navigation store. Tests set `window.location.hash` and dispatch synthetic `hashchange` events, then assert the store's reactive properties. Fourteen tests across two `describe` blocks:

**Core routing:**

1. **`defaults to 'dashboard' when hash is empty`** — asserts `currentView` is `"dashboard"` and `subPath` is `""` on load.
2. **`parses 'books' from hash`** — sets `#books`; asserts `currentView` is `"books"` with empty `subPath`.
3. **`parses 'my-library' from hash`** — sets `#my-library`; asserts `currentView` is `"my-library"`.
4. **`parses 'settings' from hash`** — sets `#settings`; asserts `currentView` is `"settings"`.
5. **`defaults invalid hash segment to 'dashboard'`** — sets `#invalid-page`; asserts `currentView` falls back to `"dashboard"`.
6. **`extracts subPath from hash`** — sets `#settings/account`; asserts `currentView` is `"settings"` and `subPath` is `"account"`.
7. **`handles multi-segment subPath`** — sets `#settings/oidc/config`; asserts `subPath` is `"oidc/config"`.
8. **`navigate sets window.location.hash`** — calls `routerStore.navigate("books")`; asserts `window.location.hash` becomes `"#books"`.
9. **`responds to hashchange events`** — dispatches a `hashchange` event after changing the hash; asserts `routerStore.hash` updates reactively.
10. **`handles hash with leading slash`** — sets `#/books`; asserts the leading slash is stripped and `currentView` is `"books"`.

**`pageTitle` sub-suite:**

11. **Parameterised title tests** — asserts the correct page title string for each of the eleven known hash values (e.g. `#dashboard` → `"Dashboard – biblioteka"`, `#settings/account` → `"Account Settings – biblioteka"`).
12. **`falls back to 'Settings – biblioteka' for unknown settings sub-path`** — sets `#settings/unknown`; asserts `pageTitle` returns the top-level settings title.
13. **`falls back to 'biblioteka' for invalid hash`** — sets `#invalid-page`; asserts `pageTitle` is just `"biblioteka"`.

### `auth.test.ts`

`frontend/src/stores/auth.test.ts` exercises `authStore`, the authentication state store. All `api.*` calls are replaced with Vitest mocks so tests run without a real backend. Fourteen tests across four `describe` blocks:

**`init` (nine tests) — application startup authentication flow:**

1. **`sets loading to false when no token and no cookie`** — no token in localStorage, `getMe()` returns 401; asserts `loading` is `false` and `user` is `null`.
2. **`fetches user when token exists`** — token in localStorage, `getMe()` resolves; asserts `user` is populated and `loading` is `false`.
3. **`clears token and retries when getMe returns 401 with token`** — stale token case; asserts `clearToken()` is called and `getMe()` is called twice.
4. **`clears stale token and authenticates via cookie on retry`** — first `getMe()` rejects with 401, second resolves with an OIDC user; asserts the user is set from the cookie session.
5. **`preserves token on transient network error`** — `getMe()` rejects with a generic `Error` (not `ApiError`); asserts `clearToken()` is **not** called.
6. **`authenticates via cookie on plain reload without token or URL marker`** — no token, no query params, `getMe()` resolves; asserts the user is set (HttpOnly cookie path).
7. **`authenticates via cookie after OIDC redirect`** — `?oidc_login=1` in the URL; asserts `getMe()` is called and the URL marker is cleaned up with `history.replaceState`.
8. **`sets oidcLinkError from URL params`** — `?oidc_link_error=account_already_linked`; asserts `authStore.oidcLinkError` is set.
9. **`redirects to settings on oidc_linked param`** — `?oidc_linked=true`; asserts `history.replaceState` redirects to `/#settings`.

**`signIn` (two tests):** asserts user is populated on success; asserts error is returned and user stays `null` on failure.

**`signUp` (two tests):** asserts user is populated on success; asserts error is returned on failure.

**`signOut` (one test):** asserts `clearToken()` and `logout()` are called and `user` is set to `null`.

> **Mocking note:** The `beforeEach` block manually resets `authStore.user`, `authStore.loading`, and `authStore.oidcLinkError` before each test. `window.location` is redefined as a writable property so query-param tests can control the URL without triggering real navigation.

### `api.test.ts`

`frontend/src/lib/api.test.ts` exercises the centralised API client. Because `api.ts` is now a barrel re-export of `frontend/src/lib/api/` sub-modules, the tests import from the barrel and exercise each sub-module through it. `fetch` is replaced with a Vitest stub so no real HTTP requests are made. Tests are grouped into nine `describe` blocks:

**`Token management` (five tests):** covers `setToken`, `clearToken`, `hasToken` — verifying `localStorage` read/write semantics, including the edge case of an empty string being treated as "no token".

**`ApiError` (one test):** asserts the custom error class has the correct `name`, `message`, `status`, and prototype chain.

**`request` (seven tests):** exercises the shared `request()` helper (called indirectly through exported API functions):
- Asserts the `Authorization: Bearer <token>` header is included when a token is stored.
- Asserts the header is omitted when no token is present.
- Asserts POST requests serialize the body as JSON.
- Asserts a non-OK JSON response throws `ApiError` with the `error` field from the body.
- Asserts a non-OK plain-text response throws `ApiError` with the response body as the message.
- Asserts fallback to `statusText` when the JSON body contains no `error` field.
- Asserts graceful handling of a JSON parse failure (falls back to the raw response text as the error message).

**`Auth API functions` (six tests):** covers `signup`, `login`, `getOidcEnabled` (three edge cases: enabled, disabled, unexpected response), and `changePassword` — verifying the correct HTTP method, URL, and request body for each call.

**`Config API` (four tests):** covers `getConfigStatus`, `getOidcConfig`, `setOidcConfig`, and `createOidcLinkNonce`.

**`Admin API` (two tests):** covers `listUsers` and `setUserAdmin`.

**`Audit Logs API` (two tests):** covers `getAuditLogs` — verifying the default `limit=50&offset=0` query string and that custom limit/offset values are forwarded correctly.

**`OPDS Credentials API` (three tests):** covers `getOpdsCredential` (GET), `setOpdsCredential` (PUT with request body), and `deleteOpdsCredential` (DELETE resolves `void` on `204 No Content`).

**`KOSync Credentials API` (three tests):** covers `getKosyncCredential` (GET), `setKosyncCredential` (PUT with request body), and `deleteKosyncCredential` (DELETE resolves `void` on `204 No Content`).

### `clipboard.test.ts`

`frontend/src/lib/clipboard.test.ts` exercises the `copyToClipboard` utility. Tests stub `navigator` and `document.execCommand` to isolate the function from real browser APIs. Tests are grouped in one `describe` block:

**`copyToClipboard` (four tests):**
- Asserts the async Clipboard API (`navigator.clipboard.writeText`) is used when available, and called with the correct text.
- Asserts the `execCommand` fallback path is taken when `navigator.clipboard` is absent, and that `document.execCommand('copy')` is invoked.
- Asserts an `Error` is thrown when `execCommand` returns `false`.
- Asserts errors thrown by the async Clipboard API are propagated to the caller.

> **Mocking note:** `beforeEach` captures the original `execCommand` property descriptor, and `afterEach` restores it alongside `vi.unstubAllGlobals()` and `vi.restoreAllMocks()`. This prevents global state leaking between tests.

---

### `libraries.test.ts`

`frontend/src/stores/libraries.test.ts` exercises `libraryStore`, the library state and scanning store. All `api.*` calls are replaced with Vitest mocks. Ten tests across five `describe` blocks:

**`load` (three tests):**
1. **`fetches libraries and sets loaded`** — mocks `listLibraries` to resolve with a fixture; asserts `libraries` is populated, `loaded` is `true`, and `loading` is `false`.
2. **`does not call API again after already loaded`** — calls `load()` twice; asserts `listLibraries` is called exactly once.
3. **`resets loading on API error`** — mocks `listLibraries` to reject; asserts `loading` and `loaded` are both `false` and `libraries` is empty.

**`add` (three tests):**
1. **`adds the created library to the store`** — mocks `createLibrary`; asserts the returned library is appended to `libraries`.
2. **`marks the newly added library as scanning`** — asserts `scanningIds` contains the new library's ID immediately after `add()`, and `isScanning` is `true`.
3. **`auto-clears scanning state after timeout`** — uses fake timers; asserts `scanningIds` is emptied after the 5-minute auto-clear timeout fires.

**`clearScanning` (two tests):**
1. **`removes the specified library ID from scanningIds`** — marks a library as scanning then calls `clearScanning(id)`; asserts `scanningIds` no longer contains it.
2. **`is a no-op when ID is not in scanningIds`** — calls `clearScanning` with an unknown ID; asserts no error and state is unchanged.

**`clearAllScanning` (one test):**
1. **`clears all scanning IDs`** — marks two libraries as scanning then calls `clearAllScanning()`; asserts `scanningIds` is empty and `isScanning` is `false`.

**`remove` (one test):**
1. **`removes the library from the store and clears its scanning state`** — adds and marks a library as scanning, then calls `remove(id)`; asserts the library is gone from `libraries` and `scanningIds`.

> **Setup:** `beforeEach` resets the store by setting `libraries = []`, `loading = false`, `loaded = false`, and calling `clearAllScanning()`. Fake timers are used to control the 5-minute scanning timeout without real waits.

---

### `BookList.test.ts`

`frontend/src/components/ui/BookList.test.ts` exercises `BookList.svelte`'s polling and empty-state behaviour. All `lucide-svelte` icons are mocked as no-ops (required for JSDOM). Six tests across two `describe` blocks:

**`BookList empty state` (two tests):**
1. **`shows 'No books yet.' when no pollingInterval is set`** — renders with an empty `fetchBooks`; asserts the standard empty-state text appears.
2. **`shows 'Scanning library...' when pollingInterval is set and no books found`** — renders with `pollingInterval={3000}` and empty `fetchBooks`; asserts the scanning spinner text appears and the standard empty text does not.

**`BookList polling` (four tests):**
1. **`polls at the specified interval when total is 0`** — uses fake timers; advances time by 1 s (twice) and asserts `fetchBooks` is called again after each step (silent poll).
2. **`stops polling once books are found`** — `fetchBooks` returns books after the first poll; asserts polling stops and `fetchBooks` is not called again after further timer advancement.
3. **`calls onBooksFound when books appear for the first time`** — passes an `onBooksFound` spy; asserts it is called exactly once when books first appear.
4. **`does not poll when no pollingInterval is set`** — renders without `pollingInterval`; advances time and asserts `fetchBooks` is called only once (initial load).

> **Mocking note:** `afterEach(cleanup)` prevents DOM leakage between tests. Fake timers (`vi.useFakeTimers()`) are activated per-suite and restored in `afterEach` to avoid contaminating other test files.

---

## Build configuration (`vite.config.ts`)

The frontend build is configured in `frontend/vite.config.ts`. Key settings:

| Setting | Value | Purpose |
|---------|-------|---------|
| `build.outDir` | `../internal/server/dist` | Output is written into the Go module so it can be embedded in the binary |
| `build.emptyOutDir` | `true` | Clears the output directory before each build to avoid stale assets |
| `server.proxy["/api"]` | `http://localhost:8080` | Forwards API requests to the Go backend during `pnpm run dev` |
| `test.environment` | `jsdom` | Vitest tests run in a browser-like DOM environment |

### `restoreGitkeep` plugin

`internal/server/dist/` is tracked by Git (via a `.gitkeep` file) so that the Go `//go:embed` directive always has a valid directory to embed — even on a clean checkout before the frontend has been built. Vite's `emptyOutDir: true` deletes the entire directory contents on each build, which would remove `.gitkeep` and break the embed on the next clean checkout.

The custom `restoreGitkeep()` Vite plugin, defined at the top of `vite.config.ts`, runs after the bundle is written (`closeBundle` hook) and recreates the empty `.gitkeep` file. This keeps Git happy without requiring a manual post-build step.

> **Do not remove** the `restoreGitkeep()` call from the `plugins` array. Without it, `go build` will fail on any checkout where the frontend has not been built, because the embedded directory will be missing.

## Linting and type-checking

```bash
# From the frontend/ directory
pnpm run lint     # ESLint
pnpm run check    # svelte-check (TypeScript + Svelte template types)
pnpm run test     # Vitest unit tests
pnpm run format   # Prettier
```

Run `lint` and `check` before every commit.
