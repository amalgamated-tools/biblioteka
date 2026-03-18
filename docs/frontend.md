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
    App.svelte          Root component: auth gate + shell layout + routing; includes skip-to-main-content link (WCAG 2.4.1)
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
      settings/           Tab sub-components for the Settings page (five of six tabs; the SMTP admin tab is inline in Settings.svelte — see Settings component architecture below)
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
  vite.config.ts      Vite configuration: build output, dev proxy, Vitest setup, and the restoreGitkeep plugin
```

## Reactive stores

All global state is managed through **Svelte 5 reactive class stores** in `frontend/src/stores/`. Each store is a class whose properties are declared with `$state` (for scalar values and nullable objects) or `$state.raw` (for array properties that are replaced wholesale on fetch), and a singleton instance is exported for use throughout the application.

### Pattern

```ts
// frontend/src/stores/example.svelte.ts
import type { Foo } from "../types";
import * as api from "../lib/api";

class ExampleStore {
  // $state.raw for arrays: tracks only the reference, not contents.
  // Avoids deep-proxy overhead and suppresses Svelte's mutation warnings
  // when the array is replaced wholesale.
  items: Foo[] = $state.raw([]);
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
5. Add a navigation `<button>` in `Settings.svelte`'s sidebar `<nav>`, wrapped in `{#if isAdmin}` if the tab is admin-only.
6. Update the table above.

## Accessibility

Biblioteka's frontend follows [WCAG 2.1](https://www.w3.org/TR/WCAG21/) guidelines. This section documents the accessibility patterns used in the app shell and how to maintain them when making changes.

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

### ARIA landmarks

The app shell uses semantic HTML5 landmark elements so screen readers can navigate by region:

| Landmark | Element | Notes |
|----------|---------|-------|
| Main navigation | `<aside>` (inside `Sidebar.svelte`) | Desktop persistent sidebar |
| Primary content | `<main id="main-content">` | Target of the skip link |
| Mobile header | `<div>` + hamburger `<button>` | Not a landmark; sits above `<main>` only on small screens |

### `autocomplete` attributes on form inputs

**WCAG criterion:** [1.3.5 Identify Input Purpose](https://www.w3.org/WAI/WCAG21/Understanding/identify-input-purpose.html) (Level AA, reaffirmed in WCAG 2.2)

The `autocomplete` attribute tells browsers and password managers exactly what kind of data an input expects. Without it, autofill heuristics may fail, and assistive technologies cannot announce the field purpose reliably.

**Password change form in `AccountTab.svelte`:**

```svelte
<input
  id="current-password"
  type="password"
  bind:value={currentPassword}
  autocomplete="current-password"
  ...
/>

<input
  id="new-password"
  type="password"
  bind:value={newPassword}
  autocomplete="new-password"
  ...
/>

<input
  id="confirm-password"
  type="password"
  bind:value={confirmPassword}
  autocomplete="new-password"
  ...
/>
```

| Input | `autocomplete` value | Why |
|-------|----------------------|-----|
| Current password | `current-password` | Lets password managers fill in the existing credential |
| New password | `new-password` | Signals to password managers and browsers to offer a *generate* option, not autofill the current one |
| Confirm new password | `new-password` | Matches the new-password intent; browsers typically suppress fill-in here automatically |

**Rule for all new password inputs:** always set `autocomplete` to the appropriate [HTML autofill token](https://html.spec.whatwg.org/multipage/form-control-infrastructure.html#autofilling-form-controls:-the-autocomplete-attribute). For other personal-data inputs, consult the full autofill token list (e.g., `email`, `name`, `username`, `one-time-code`).

### Maintaining accessibility

When editing the app shell or adding new persistent navigation elements:

1. Keep the skip link as the **first** child of the authenticated shell `<div>`.
2. If you add a new persistent region that users must bypass, add an additional skip link or update the existing one.
3. All interactive elements that are not natively focusable must have `tabindex="-1"` (receive focus programmatically only) or `tabindex="0"` (enter the natural tab order). Never use `tabindex` values greater than `0`.
4. Always set `autocomplete` on inputs that collect personal data or credentials (see the section above).
5. Run `pnpm run check` — `svelte-check` will surface missing `alt` attributes and other common issues.

### Accessibility tests

`frontend/src/App.test.ts` contains a focused regression test that verifies:

- The skip link exists and has the label _"Skip to main content"_.
- The `<main>` landmark has `id="main-content"`.
- The skip link appears **before** the sidebar in DOM order.
- Clicking the skip link moves keyboard focus to `<main>`.

Keep this test green. If you restructure `App.svelte`, update the test to match.

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
