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
      Books.svelte        Book listing and detail view
      Dashboard.svelte    Home screen; library overview
      Libraries.svelte    Library management view
      MyLibrary.svelte    Placeholder for a planned per-user personal library feature; currently shows an empty state
      Settings.svelte     Settings shell; owns shared admin state; renders one tab at a time
      Sidebar.svelte      Navigation sidebar; fetches and displays the running server version; uses `<a href>` anchor links for all navigation items; icon-only action links (Create library, Library settings) carry `aria-label` and `aria-hidden="true"` on their icons (WCAG 4.1.2)
      libraries/          Reusable sub-components for the Libraries view
        LibraryForm.svelte   Create / edit library form
        LibraryView.svelte   Library detail with book listing
      settings/           Tab sub-components for the Settings page (see Settings component architecture below)
        AccountTab.svelte       Account & password management; OIDC linking
        APIKeysTab.svelte       Create and revoke long-lived API keys (`bib_` prefix)
        KoboTab.svelte          Kobo sync token management; displays setup instructions
        OidcTab.svelte          Admin: OIDC / SSO provider configuration
        PreferencesTab.svelte   Display theme selection
        SmtpTab.svelte          Admin: SMTP mail server configuration
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
| `pageTitle` | `string` | Human-readable page title for the current view (e.g. `"Dashboard – biblioteka"`); used to update `document.title` on every navigation |
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
| `KoboTab.svelte` | `settings/kobo` | All users | Create and revoke Kobo sync tokens; copy device sync URL |
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

<!-- Panels: one per tab -->
<div id="login-panel"  role="tabpanel" tabindex="0" aria-labelledby="login-tab"  hidden={!isLogin}>
  <!-- login form -->
</div>
<div id="signup-panel" role="tabpanel" tabindex="0" aria-labelledby="signup-tab" hidden={isLogin}>
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
| `tabindex="0"` | Each panel | Makes the panel itself focusable so keyboard users can reach its content |
| `hidden` | Inactive panel | Hides the inactive panel from both display and the accessibility tree |

**Roving tabindex:**
Only the active tab sits in the natural tab order (`tabindex="0"`); inactive tabs are removed from it (`tabindex="-1"`) but remain focusable programmatically. This means `Tab` enters the tab bar once and then moves directly to the active panel's content — inactive tabs are skipped, matching the expected [APG tab pattern](https://www.w3.org/WAI/ARIA/apg/patterns/tabs/) behaviour.

**Keyboard navigation (`handleTabKeydown`):**

| Key | Action |
|-----|--------|
| `ArrowRight` / `ArrowLeft` | Move focus to the next / previous tab and activate it |
| `Home` | Move focus and activation to the first tab (Login) |
| `End` | Move focus and activation to the last tab (Sign Up) |

**Why `hidden` instead of Svelte `{#if}`:**
The `hidden` HTML attribute is used on inactive panels rather than Svelte's `{#if}` block. Both panels stay in the DOM, so `aria-controls` references always point to a valid element. Removing a panel with `{#if}` would leave a dangling `aria-controls` reference and break the ARIA association.

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
5. Every icon-only link or button must have `aria-label`; the icon element inside the control must carry `aria-hidden="true"` to suppress redundant announcements; every unlabelled input must have `aria-label` or `aria-labelledby`. See [Accessible labels for icon-only controls](#accessible-labels-for-icon-only-controls) above.
6. Navigation links that represent the active view must carry `aria-current={isActive ? "page" : undefined}`. Tab-style buttons should use `aria-selected` instead (see item 8). See [`aria-current` on active navigation links](#aria-current-on-active-navigation-links) above.
7. Toggle switches (`<input type="checkbox">` styled as a switch) must carry `role="switch"`. See [`role="switch"` on toggle inputs](#roleswitch-on-toggle-inputs) below.
8. Tab-style navigation widgets (a set of buttons that show/hide panels) must use the ARIA tablist/tab/tabpanel pattern with roving tabindex and keyboard navigation (Arrow keys, Home, End). See [ARIA tab widget — Login/Sign Up toggle](#aria-tab-widget--loginsign-up-toggle-authsvelte) for the reference implementation.
9. Data tables must have `scope="col"` (or `scope="row"`) on every `<th>`. Visual-only columns (e.g., "Actions") must have an `sr-only` text label inside their `<th>`. See [Table accessibility](#table-accessibility) below.
10. Run `pnpm run check` — `svelte-check` will surface missing `alt` attributes and other common issues.

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

Add `role="switch"` to any `<input type="checkbox">` that renders as a toggle:

```svelte
<input
  type="checkbox"
  role="switch"
  bind:checked={formMonitored}
  class="sr-only peer"
/>
```

With `role="switch"`, assistive technologies announce the control as a _switch_ and report its state as _on_ or _off_ (rather than _checked_ or _unchecked_), which is the semantically correct announcement for a toggle.

**Guidelines:**

- Apply `role="switch"` whenever a `<input type="checkbox">` is styled to look like a toggle switch.
- The control must still have an accessible name — either via a paired `<label>` element (preferred) or `aria-label`.
- Do **not** use `role="switch"` on checkboxes that genuinely represent a tri-state or multi-select option; use plain `type="checkbox"` there.

#### Checklist for new forms

When adding or editing a form component:

1. Every `<input>`, `<select>`, and `<textarea>` has either a `<label for="…">` or an `aria-label`.
2. `<label for>` values match the corresponding `id` exactly — a mismatch silently breaks the association.
3. Related inputs that share a common heading are wrapped in a `<fieldset>` with a `<legend>`. See [`fieldset` and `legend` for grouped inputs](#fieldset-and-legend-for-grouped-inputs) above.
4. Icon-only buttons (`<button>` with SVG content and no text) carry an `aria-label`.
5. Repeated inputs in `{#each}` blocks use a dynamic, positionally-distinct `aria-label`.
6. Inputs with inline validation errors set `aria-invalid="true"` and `aria-describedby="<error-id>"` when the error is visible. See [`aria-describedby` for inline validation errors](#aria-describedby-for-inline-validation-errors) above.
7. Password inputs (`type="password"`) carry `autocomplete="current-password"` or `autocomplete="new-password"` as appropriate.
8. Toggle switches (`<input type="checkbox">` styled as a switch) carry `role="switch"`.
9. Tab-style widgets that show/hide panels use the ARIA tablist/tab/tabpanel pattern with roving tabindex, `aria-selected`, `aria-controls`/`aria-labelledby`, and keyboard navigation (Arrow keys, Home, End). See [ARIA tab widget — Login/Sign Up toggle](#aria-tab-widget--loginsign-up-toggle-authsvelte) for the reference implementation.
10. Run `pnpm run check` after your changes — `svelte-check` will catch missing `alt` on images and some label issues.

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

### Accessibility tests

Accessibility regressions are locked in by dedicated test files. Keep all of these tests green. If you restructure a component, update its test to match.

#### `App.test.ts`

`frontend/src/App.test.ts` verifies the authenticated shell:

- `document.title` is set from `routerStore.pageTitle` on mount (WCAG 2.4.2).
- The skip link exists and has the label _"Skip to main content"_.
- The `<main>` landmark has `id="main-content"`.
- The skip link appears **before** the sidebar in DOM order.
- Clicking the skip link moves keyboard focus to `<main>`.

#### `Auth.test.ts`

`frontend/src/components/Auth.test.ts` verifies the ARIA tab widget on the login/signup page (WCAG 4.1.2). It uses `@testing-library/user-event` to simulate real browser interactions and `await tick()` to flush Svelte 5 reactivity before asserting. Three tests are included:

1. **`has a main landmark region`** — asserts the login page exposes a `<main>` landmark (WCAG 1.3.6).
2. **`renders tab buttons with correct ARIA attributes`** — asserts the `role="tablist"` container is present and that each tab carries the correct `aria-selected` and `aria-controls` attributes on initial render.
3. **`renders tab panels with correct ARIA attributes`** — asserts both `role="tabpanel"` elements (including the hidden one) carry the correct `aria-labelledby` back-reference.
4. **`switches ARIA state when the Sign Up tab is clicked`** — simulates a user click on the Sign Up tab and verifies that `aria-selected` values update reactively and the Sign Up panel loses its `hidden` attribute.

> **Testing note:** Each test wraps `render(Auth)` in an `async renderAuth()` helper that calls `await tick()` after mounting. This is necessary because Svelte 5 defers reactive updates; without the tick the DOM may not reflect the initial `$state` values when the first `expect` runs. The `afterEach(cleanup)` guard ensures the JSDOM is cleared between tests to prevent state bleed.

#### `Sidebar.test.ts`

`frontend/src/components/Sidebar.test.ts` verifies that the navigation sidebar uses semantic `<a>` anchor links (not `<button>` elements) for in-app navigation, and that `aria-current` state is applied correctly (WCAG 4.1.2). Four tests are included:

1. **`renders Dashboard, All Books, and Settings as links with correct hrefs`** — asserts that the three primary nav items are rendered as `role="link"` elements with the correct hash `href` values (`#dashboard`, `#books`, `#settings`).
2. **`renders Logout as a button, not a link`** — asserts that Logout is a `role="button"` element, which is semantically correct because it triggers an action (sign-out) rather than navigating to a URL.
3. **`sets aria-current='page' on the active navigation link`** — renders with `currentView="settings"` and asserts that the Settings link carries `aria-current="page"` while Dashboard does not.
4. **`sets aria-current='page' on the active library link`** — renders with `currentView="libraries"` and `subPath="1"` and asserts that only the matching library entry link receives `aria-current="page"`.

> **Mocking note:** The test file mocks `authStore`, `libraryStore`, `api.getVersion`, and all `lucide-svelte` icon components. The icon mocks are necessary because Lucide icons are ESM-only packages that cannot render in JSDOM; replacing them with no-ops keeps the test focused on DOM structure. `afterEach(cleanup)` prevents DOM leakage between tests.

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
