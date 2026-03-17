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
    App.svelte          Root component: auth gate + shell layout + routing
    main.ts             Entry point; mounts App and initialises the theme
    index.css           Tailwind CSS directives
    types.ts            Shared TypeScript interfaces for API entities
    components/         Page-level Svelte components (PascalCase)
      Auth.svelte         Login and signup forms
      Books.svelte        Book listing and detail view
      Dashboard.svelte    Home screen; library overview
      Libraries.svelte    Library management view
      MyLibrary.svelte    Placeholder for a planned per-user personal library feature; currently shows an empty state
      Settings.svelte     Settings shell; owns shared admin state; renders one tab at a time
      Sidebar.svelte      Navigation sidebar; fetches and displays the running server version
      libraries/          Reusable sub-components for the Libraries view
        LibraryForm.svelte   Create / edit library form
        LibraryView.svelte   Library detail with book listing
      settings/           Sub-components for the Settings page (one per tab)
        AccountTab.svelte       Account & password management; OIDC linking
        APIKeysTab.svelte       Create and revoke long-lived API keys (`bib_` prefix)
        OidcTab.svelte          Admin: OIDC / SSO provider configuration
        PreferencesTab.svelte   Display theme selection
        UsersTab.svelte         Admin: user list and admin-role toggling
      ui/                 Generic reusable UI components
        AlertBanner.svelte   Dismissible alert / error banner
        BookCard.svelte      Card widget displaying a single book summary
        BookList.svelte      Paginated book list with grid / table view toggle; accepts a `fetchBooks` callback
    stores/             Reactive state modules (lowercase, *.svelte.ts)
    lib/
      api.ts            Centralised API client
      api.test.ts       API client unit tests
```

## Reactive stores

All global state is managed through **Svelte 5 reactive class stores** in `frontend/src/stores/`. Each store is a class whose properties are declared with `$state` (for scalar values and nullable objects) or `$state.raw` (for array collections), and a singleton instance is exported for use throughout the application.

### Pattern

```ts
// frontend/src/stores/example.svelte.ts
import type { Foo } from "../types";
import * as api from "../lib/api";

class ExampleStore {
  items: Foo[] = $state.raw([]);   // $state.raw for arrays — avoids deep-proxy overhead and suppresses Svelte's mutation warnings when the array is replaced wholesale
  loading = $state(false);
  loaded  = $state(false);

  async load(): Promise<void> { … }
  async add(input: FooInput): Promise<Foo> { … }
  async edit(id: string, input: FooInput): Promise<Foo> { … }
  async remove(id: string): Promise<void> { … }
}

export const exampleStore = new ExampleStore();
```

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
| `libraries.svelte.ts` | `libraryStore` | Library CRUD; cached after first load |
| `authors.svelte.ts` | `authorStore` | Author CRUD; cached after first load |
| `series.svelte.ts` | `seriesStore` | Series CRUD; cached after first load |

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

## Routing

Client-side routing uses the browser's URL hash (`#`). No router library is needed.

`routerStore` exposes:

| Property | Type | Description |
|----------|------|-------------|
| `hash` | `string` | Raw hash value (e.g. `"settings/account"`) |
| `currentView` | `AppView` | Top-level view segment (`"dashboard"` \| `"books"` \| `"my-library"` \| `"libraries"` \| `"settings"`) |
| `subPath` | `string` | Sub-path after the first segment (e.g. `"account"`) |
| `navigate(path)` | `void` | Sets the hash and updates the store |

**Navigating programmatically:**

```ts
import { routerStore } from "../stores/router.svelte";

routerStore.navigate("settings/account");
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

All HTTP calls go through `frontend/src/lib/api.ts`. This module:

- Stores the JWT token in `localStorage` and attaches it as the `Authorization: Bearer` header on every authenticated request.
- Throws a typed `Error` with the server's `error` message on non-2xx responses.
- Exports a function per API resource (e.g. `listBooks`, `createBook`, `updateBook`, `deleteBook`).

```ts
// Example usage inside a store method
import * as api from "../lib/api";

const book = await api.createBook({ title: "Dune", … });
```

Never call `fetch` directly from components or stores — always go through `api.ts`.

## TypeScript types

TypeScript types are split between two files based on their purpose:

| File | What goes here |
|------|----------------|
| `frontend/src/types.ts` | **Domain entity types** — interfaces for API resource models that are used across multiple components and stores (e.g. `Library`, `Author`, `Book`, `BookFile`). |
| `frontend/src/lib/api.ts` | **API-specific types** — request/response shapes tightly coupled to a single API module group (e.g. `ConfigStatus`, `OIDCConfig`, `SetOIDCConfigInput`, `AdminUser`). These may be imported into components that use the relevant API functions. |

Never inline types directly in `.svelte` component files or `*.svelte.ts` store files. If a type is shared across more than one component or store, move it to `types.ts`.

## Adding a new store

1. Create `frontend/src/stores/<name>.svelte.ts`.
2. Define a class with `$state` / `$state.raw` properties. Use `$state.raw` for array properties and `$state` for scalars and nullable objects (see the [`$state` vs `$state.raw`](#reactive-stores) note above).
3. Export a singleton: `export const myStore = new MyStore();`.
4. Add an entry for the new store in the table above.

## Adding a new view

1. Create `frontend/src/components/MyView.svelte`.
2. Add the new route identifier to the `AppView` union type in `router.svelte.ts`.
3. Add the route to the `valid` array in `RouterStore.currentView`.
4. Import and render `<MyView />` in `App.svelte` inside the `{#if … }` routing block.
5. Add a navigation entry in `Sidebar.svelte`.

## UI components

The three reusable components in `frontend/src/components/ui/` are shared across multiple page-level components. They accept only typed props — no global store access — and are safe to use in any context.

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

**Pagination behaviour:**

- On mount and whenever `fetchBooks` or `pageSize` changes, `offset` resets to `0` and a fresh fetch is triggered.
- If items are deleted and the current page becomes empty (but earlier pages still have items), `BookList` automatically clamps back to the last valid page.
- Stale responses from superseded fetches are silently discarded via an internal request-ID counter.

---

## Settings component architecture

`Settings.svelte` is a shell that owns shared state (admin flag, OIDC config) and renders one tab at a time. Each tab is a standalone sub-component in `frontend/src/components/settings/`.

| Component | Route | Visibility | Responsibility |
|-----------|-------|------------|----------------|
| `AccountTab.svelte` | `settings/account` | All users | Change password; link OIDC account |
| `APIKeysTab.svelte` | `settings/api-keys` | All users | Create and revoke long-lived API keys (`bib_` prefix) |
| `PreferencesTab.svelte` | `settings/preferences` | All users | Choose light / dark / auto theme |
| `OidcTab.svelte` | `settings/oidc` | Admins only | Configure OIDC / SSO provider |
| *(inline in `Settings.svelte`)* | `settings/smtp` | Admins only | Configure SMTP mail server |
| `UsersTab.svelte` | `settings/users` | Admins only | List users; toggle admin role |

`Settings.svelte` passes data down as props and receives updates via callback props (`onOidcSaved`, `onUsersLoaded`), keeping each tab stateless with respect to shared data. The SMTP tab is the exception: its state and logic live directly in `Settings.svelte` rather than in a dedicated sub-component.

### Adding a new settings tab

1. Create `frontend/src/components/settings/MyTab.svelte`.
2. Define an `interface Props { … }` and use `$props()` for any data the tab needs from `Settings.svelte`.
3. Add `"my-tab"` to the `SettingsTab` union type and `validTabs` array in `Settings.svelte`.
4. Import and render `<MyTab />` inside the `{#if activeTab === "my-tab"}` block in `Settings.svelte`.
5. Add a navigation `<button>` in `Settings.svelte`'s sidebar `<nav>`, wrapped in `{#if isAdmin}` if the tab is admin-only.
6. Update the table above.

## Linting and type-checking

```bash
# From the frontend/ directory
pnpm run lint     # ESLint
pnpm run check    # svelte-check (TypeScript + Svelte template types)
pnpm run test     # Vitest unit tests
pnpm run format   # Prettier
```

Run `lint` and `check` before every commit.
