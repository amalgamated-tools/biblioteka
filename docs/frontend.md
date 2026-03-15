# Frontend Architecture

Biblioteka's frontend is a single-page application (SPA) built with **Svelte 5**, **TypeScript**, and **Tailwind CSS 3**. It is bundled by **Vite** and served as static assets embedded in the Go binary.

## Directory layout

```
frontend/src/
  App.svelte          Root component: auth gate + shell layout + routing
  main.ts             Entry point; mounts App and initialises the theme
  index.css           Tailwind CSS directives
  types.ts            Shared TypeScript interfaces for API entities
  components/         Page-level Svelte components (PascalCase)
    settings/         Sub-components for the Settings page (one per tab)
      AccountTab.svelte    Account & password management; OIDC linking
      OidcTab.svelte       Admin: OIDC / SSO provider configuration
      PreferencesTab.svelte Display theme selection
      UsersTab.svelte      Admin: user list and admin-role toggling
  stores/             Reactive state modules (lowercase, *.svelte.ts)
  lib/
    api.ts            Centralised API client
    api.test.ts       API client unit tests
```

## Reactive stores

All global state is managed through **Svelte 5 reactive class stores** in `frontend/src/stores/`. Each store is a class whose properties are declared with `$state` (or `$state.raw` for array collections), and a singleton instance is exported for use throughout the application.

### Pattern

```ts
// frontend/src/stores/example.svelte.ts
import type { Foo } from "../types";
import * as api from "../lib/api";

class ExampleStore {
  items: Foo[] = $state.raw([]);   // $state.raw for arrays — avoids deep reactivity overhead
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

### Available stores

| File | Export | Purpose |
|------|--------|---------|
| `auth.svelte.ts` | `authStore` | Current user, sign-in/up/out, OIDC token initialisation |
| `router.svelte.ts` | `routerStore` | Hash-based navigation; current view and sub-path |
| `theme.svelte.ts` | `themeStore` | Light / dark / auto theme preference; persisted to `localStorage` |
| `libraries.svelte.ts` | `libraryStore` | Library CRUD; cached after first load |
| `books.svelte.ts` | `bookStore` | Book CRUD; cached after first load |
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
2. Define a class with `$state` / `$state.raw` properties.
3. Export a singleton: `export const myStore = new MyStore();`.
4. Add an entry for the new store in the table above.

## Adding a new view

1. Create `frontend/src/components/MyView.svelte`.
2. Add the new route identifier to the `AppView` union type in `router.svelte.ts`.
3. Add the route to the `valid` array in `RouterStore.currentView`.
4. Import and render `<MyView />` in `App.svelte` inside the `{#if … }` routing block.
5. Add a navigation entry in `Sidebar.svelte`.

## Settings component architecture

`Settings.svelte` is a shell that owns shared state (admin flag, OIDC config) and renders one tab at a time. Each tab is a standalone sub-component in `frontend/src/components/settings/`.

| Component | Route | Visibility | Responsibility |
|-----------|-------|------------|----------------|
| `AccountTab.svelte` | `settings/account` | All users | Change password; link/unlink OIDC account |
| `PreferencesTab.svelte` | `settings/preferences` | All users | Choose light / dark / auto theme |
| `OidcTab.svelte` | `settings/oidc` | Admins only | Configure OIDC / SSO provider |
| `UsersTab.svelte` | `settings/users` | Admins only | List users; toggle admin role |

`Settings.svelte` passes data down as props and receives updates via callback props (`onOidcSaved`, `onUsersLoaded`), keeping each tab stateless with respect to shared data.

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
