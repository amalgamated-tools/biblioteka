# Glossary

Project-specific terminology used across Biblioteka's codebase and documentation.

---

## `$derived`

A Svelte 5 rune that declares a computed value recalculated whenever its reactive dependencies change. Used throughout the frontend for values derived from store state (e.g. `let scanning = $derived(libraryStore.scanningIds.has(libraryId))`). Replaces Svelte 4's `$: derivedValue = ...` reactive statements. See also [`$state`](#state), [runes](#runes).

## `$effect`

A Svelte 5 rune that runs a side effect whenever its reactive dependencies change. Preferred over `onMount` for one-time data fetching in runes-mode components, because `CrudStore.load()` includes an idempotency guard that prevents duplicate requests even if the effect re-runs. For initialization that must run exactly once regardless of reactive dependencies, guard with a non-reactive boolean. See also [runes](#runes), [CrudStore](#crudstore).

## `$props`

A Svelte 5 rune that declares a component's typed props. Replaces Svelte 4's `export let` prop declarations (e.g. `let { libraryId }: { libraryId: string } = $props()`). See also [runes](#runes).

## `$state`

A Svelte 5 rune that declares a reactive state variable. Used for scalar values (booleans, strings, numbers) and nullable object references. Svelte creates a deep reactive proxy over the value. Use [`$state.raw`](#stateraw) instead for arrays that are replaced wholesale on every fetch. See also [runes](#runes).

## `$state.raw`

A variant of [`$state`](#state) that tracks only the reference to a value, not its contents. Used for array properties that are replaced wholesale on every fetch, avoiding unnecessary deep-proxy overhead and preventing the "state mutated outside a reactive context" console warning. See also [runes](#runes).

## API Key

A long-lived authentication token with the `bib_` prefix used for programmatic access to the Biblioteka API. API keys are generated as cryptographically random hex strings, stored by their hash (not in plaintext), and returned only once at creation time. Managed under **Settings → API Keys** or via the REST API. See [Authentication](authentication.md).

## ASIN

**Amazon Standard Identification Number.** A product identifier assigned by Amazon, present in MOBI and AZW3 files and some Kindle-converted EPUB files. Biblioteka extracts the ASIN from book file metadata but does not automatically write it to the database during import — it must be set explicitly via the API after import if needed. See [Metadata Extraction](metadata.md).

## asynq

A Redis-backed task queue for Go used by Biblioteka for background jobs such as library scanning and book file processing. Jobs are enqueued by the HTTP server (or a scheduler) and consumed by a worker process running up to four concurrent goroutines. A built-in monitoring dashboard is available at `/asynqmon/`. See [Background Jobs](background-jobs.md).

## AZW3

Amazon's Kindle Format 8 (KF8) e-book file format. Biblioteka supports AZW3 file import, metadata extraction via ExifTool, and serving AZW3 files to Kobo devices as the `AZW3` format identifier. See [Metadata Extraction](metadata.md).

## `book_per_file`

A library organization layout where imported book files are moved into `<Author>/` directories (one level deep). The book title is not required — only an author is needed. Because multiple books share the same author directory, sidecar filenames are prefixed with the book's filename stem (e.g. `gatsby.jpg`, `gatsby.opf`). See also [`book_per_folder`](#book_per_folder), [`none`](#none-organization).

## `book_per_folder`

A library organization layout where imported book files are moved into `<Author>/<Title>/` directories (two levels deep). Both an author and a title must be present after metadata extraction; if either is absent the file stays in place. See also [`book_per_file`](#book_per_file), [`none`](#none-organization).

## CrudStore

The base class in `frontend/src/stores/` from which entity-list stores inherit. Provides a `$state.raw` field for `items` (replacing the whole array on each fetch) and `$state` fields for `loading`, `loaded`, and `error`, plus a `load()` method with an idempotency guard (`if (this.loading || this.loaded) return`) that prevents duplicate API requests even when called from multiple `$effect` blocks. Stores that require additional state beyond basic CRUD (e.g. `libraryStore` with scan-tracking) implement the full class directly instead of extending `CrudStore`. See also [`$effect`](#effect), [runes](#runes).

## dbmate

The database migration tool used by Biblioteka. Migration files live in `db/migrations/sqlite/` and `db/migrations/postgres/` and run automatically on server startup. Files follow the naming convention `YYYYMMDDHHMMSS_description.sql` and use the `-- migrate:up` / `-- migrate:down` format. See [Database Schema](database-schema.md).

## ExifTool

An external command-line tool ([exiftool.org](https://exiftool.org/)) that Biblioteka uses to extract metadata (title, author, ISBN, description, publisher, language, publication date, cover art) from EPUB, MOBI, AZW3, and PDF files. ExifTool must be installed and on `PATH`. It runs as a stay-open subprocess managed by `internal/exif`. See [Metadata Extraction](metadata.md).

## Goodreads Metadata

A metadata suggestion record fetched from Goodreads and stored in the `goodreads_metadata` table. Each record belongs to a user and optionally to a specific book. Records move through a three-state lifecycle:

| Status | Meaning |
|--------|---------|
| `pending` | Fetched from Goodreads; awaiting user review |
| `applied` | User accepted the suggestion; the book record has been updated |
| `rejected` | User closed the suggestion without using the one-shot apply; the book may or may not have been updated manually |

Fetching is triggered via the API, which enqueues a background job (`enrich:goodreads`). The job searches Goodreads, creates a `pending` record, and streams progress events over SSE. The user then reviews the suggestion and chooses to apply or reject it. If a `pending` record already exists for a given `(user, book)` pair, the fetch endpoint skips enqueueing another job, and reads use the most recent pending record for that pair. The provenance identifier `"goodreads"` is included in API responses as the `source` field and in audit log entries. See also [`ASIN`](#asin).

## JWT

**JSON Web Token.** The authentication token issued by Biblioteka after a successful login or OIDC callback. The JWT is short-lived and must be supplied in the `Authorization: Bearer <token>` header for API requests. Browser clients also receive it as an HttpOnly `SameSite=Strict` cookie (`biblioteka_token`). See [Authentication](authentication.md).

## KEPUB

Kobo Enhanced EPUB — Kobo's proprietary extension of the EPUB format that enables additional reading features on Kobo devices (e.g. chapter progress, enhanced typography). Biblioteka serves KEPUB files to Kobo devices using the `KEPUB` format identifier in the sync API. See [Kobo Sync](kobo.md).

## Kobo Sync Token

A high-entropy random token that embeds the Kobo sync URL: `https://<host>/kobo/<token>`. The token authenticates a Kobo device without requiring a username and password — the device's `StoreURL` is set to this URL. Each user can create multiple tokens (one per device is recommended) so individual devices can be revoked independently. The plaintext token is shown only once at creation time. See [Kobo Sync](kobo.md).

## KOReader

An open-source e-reader application ([koreader.rocks](https://koreader.rocks/)) that runs on many e-ink devices including Kobo, Kindle, and PocketBook. Biblioteka serves KOReader via both the [OPDS](#opds) catalog (for browsing and downloading books) and the [KOSync](#kosync) API (for reading progress sync). See [KOReader Sync](koreader.md), [OPDS Catalog](opds.md).

## KOSync

Biblioteka's kosync-compatible reading-progress sync API (`/api/syncs/progress`). KOReader's **Progress sync** plugin connects to this endpoint using a dedicated username and password (separate from the main Biblioteka account). Biblioteka stores one progress record per `(user, document)` pair in the `reading_progress` table; the `document` identifier is an opaque string chosen by KOReader from the book's file hash or path. See [KOReader Sync](koreader.md).

## Library

A named collection of book files with one or more configured filesystem paths. Libraries can be set to `monitored = true` so the scheduler automatically scans them every 24 hours for new content. Each library has an `organization_type` that controls how imported files are arranged on disk. See also [`book_per_folder`](#book_per_folder), [`book_per_file`](#book_per_file), [`none`](#none-organization).

## `none` (organization)

A library organization layout that leaves imported book files in place — no files are moved or renamed. Sidecar files are still written alongside each book file. The `none` type is the default for libraries whose files should not be reorganized. See also [`book_per_folder`](#book_per_folder), [`book_per_file`](#book_per_file).

## OIDC

**OpenID Connect.** An identity layer on top of OAuth 2.0 that Biblioteka supports for single sign-on (SSO). Users can link an OIDC provider account to their Biblioteka account, after which they can log in via the provider's authentication flow. OIDC settings (issuer URL, client ID/secret) are configured by admins under **Settings → OIDC**. See [Authentication](authentication.md).

## OPDS

**Open Publication Distribution System.** A standard (version 1.2) that exposes a book library as a browsable Atom feed. Biblioteka's built-in OPDS catalog at `/opds` supports navigation, acquisition, and full-text search feeds. OPDS clients include KOReader, Calibre, Moon+ Reader, PocketBook, and Aldiko. Authentication uses HTTP Basic Auth with OPDS-specific credentials (separate from the main account). See [OPDS Catalog](opds.md).

## OPF

**Open Packaging Format.** An XML metadata format (OPF 2.0, Dublin Core) used by Biblioteka for sidecar files. When a book file is imported, Biblioteka writes a `metadata.opf` file alongside it containing title, author, identifier, language, publication date, publisher, and description. See also [sidecar files](#sidecar-files).

## Passkey

A phishing-resistant, passwordless authentication credential based on the WebAuthn Level 2 standard. A passkey is a cryptographic key pair stored on the user's device — a hardware security key, fingerprint reader, Face ID, Windows Hello, or mobile platform authenticator. After registering a passkey while logged in (at **Settings → Account → Passkeys**), the user can log in using a local biometric or PIN gesture without entering a password. Because passkeys use discoverable credentials, no username is required at login time. Password and OIDC login remain available alongside passkeys.

Passkeys require server-side configuration in non-`localhost` environments: `WEBAUTHN_RP_ID` must exactly match the domain users access (e.g. `books.example.com`); `WEBAUTHN_RP_ORIGINS` lists the allowed origins; `WEBAUTHN_RP_NAME` sets the name shown in the browser dialog. The default values only work at `http://localhost:8080`. See [Authentication](authentication.md).

## Reading Group

A collaborative space where users read together. Each reading group has one **owner** (the user who created it) and any number of **members**. Members can:

- Share their personal [reading lists](#reading-list) with the group so other members can see the shared list metadata and `book_count`
- Compare per-member reading progress for any book via `GET /api/groups/{id}/progress?book_id={book_id}` (progress values come from each member's Kobo reading data)

A user can only see groups they belong to — non-members receive `404 Not Found` to avoid leaking group existence. Group names are unique per owner after normalization. The owner cannot remove themselves from a group; to disband, they must delete the group with `DELETE /api/groups/{id}`.

Reading groups are managed via the REST API at `/api/groups` and sub-resources `/api/groups/{id}/members`, `/api/groups/{id}/lists`, and `/api/groups/{id}/progress?book_id={book_id}`. See [API Reference — Reading Groups](api/reading-groups.md).

## Reading List

A user-curated named collection of books. Reading lists are user-scoped — each user manages their own lists independently. Each list has a required name (unique per user after normalization), an optional description, and a `book_count` computed at read time. Books are added and removed individually; both operations are idempotent.

Reading lists are managed via the REST API at `/api/reading-lists` and `/api/reading-lists/{id}/books`. The `GET /api/books/{id}/reading-lists` endpoint returns the lists that contain a specific book. Frontend state is managed by `readingListStore` and the feature is accessible from the **Reading Lists** sidebar entry. See [API Reference](api-reference.md).

## Recommendations

A scored list of books the authenticated user has not yet read, generated locally without an external service. The ranking algorithm combines four signals derived from the user's Kobo reading history: author overlap with books the user is currently reading or has finished, series continuation (books in the same series as books the user is reading or has finished), publisher match, and overall download popularity across the instance. Results are returned by `GET /api/recommendations` (default 10, max 50) and displayed on the dashboard as a **You Might Also Like** panel backed by the `Recommendations.svelte` component. See [API Reference](api-reference.md).

## Runes

Svelte 5's compiler-level reactivity primitives, written as `$`-prefixed keywords (`$state`, `$derived`, `$effect`, `$props`, `$derived.by`). Runes replace Svelte 4's `writable`/`readable` store API and `$:` reactive statements with explicit, fine-grained reactive declarations. All Biblioteka frontend code uses runes mode exclusively. See also [`$state`](#state), [`$derived`](#derived), [`$effect`](#effect), [`$props`](#props).

## Sidecar Files

Two companion files written alongside every imported book file, regardless of whether file organization is enabled:

| Filename | Contents |
|----------|----------|
| `cover.jpg` | Embedded cover image extracted from the book file |
| `metadata.opf` | [OPF](#opf) 2.0 Dublin Core metadata (title, author, identifier, language, date, publisher, description) |

For `book_per_file` libraries where multiple books share the same directory, sidecar filenames are prefixed with the book's filename stem (e.g. `gatsby.jpg`, `gatsby.opf`). Sidecar files are compatible with Calibre, KOReader, and Kobo. See [Administration](administration.md).

## Tag

A user-defined label applied to books for categorization and discovery. Tags are globally-scoped named entities (not per-user) with their own CRUD API at `/api/tags`. A tag name is normalized before storage (whitespace trimmed and collapsed). Tags can be assigned to a book via `PUT /api/books/{id}/tags` and retrieved via `GET /api/books/{id}/tags`. Multiple tags can be applied to a single book, and a single tag can be applied to many books. Tags are also populated during AI metadata enrichment when using the Ollama provider. See [API Reference](api-reference.md).

## WAL

**Write-Ahead Logging.** The SQLite journal mode used by Biblioteka (`PRAGMA journal_mode=WAL`). WAL mode allows concurrent readers and a single writer without blocking reads during writes, improving performance for a multi-user web application. Combined with `synchronous=NORMAL` and `foreign_keys=ON` in all SQLite connections. See [Database Schema](database-schema.md).
