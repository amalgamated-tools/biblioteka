# Product Scan Findings

## Phase Metadata
- Lenses run: 5 (A: Error Paths, B: State Machine, C: Data Integrity, D: Boundary/QA, E: Frontend UX)
- Features/pages covered: ~40
- Raw findings before synthesis: 30+ (3 withdrawn by sub-agents)
- Final findings: 27

---

## CRITICAL

### Finding P1: "Active Days" and reading streak severely undercount for users who read one book at a time
**WHAT THE USER SEES:**
1. User reads one novel every day for 30 days.
2. Dashboard shows `1-day streak` and "1 days reading".
3. The streak/active-days metrics punish the behavior they're supposed to celebrate.

**ROOT CAUSE:** `reading_progress` has `UNIQUE(user_id, document)`. `UpsertReadingProgress` overwrites `updated_at` on every sync. So the table is current-state, not history. `GetYearInBooks.ActiveDays = COUNT(DISTINCT DATE(updated_at))` counts distinct days where the LAST sync of ANY document fell — not days actually read.

**EVIDENCE:**
- `db/migrations/sqlite/20260317000001_create_kosync_tables.sql:25` — `UNIQUE (user_id, document)`.
- `internal/db/kosync.go:88-94` — `ON CONFLICT (user_id, document) DO UPDATE SET ... updated_at = NOW()`.
- `internal/db/reading_progress.go:217-228` — counts distinct DATE(updated_at).
- Tests `internal/db/reading_progress_test.go:445-489` confirm the codepath only works with different documents per day.

**IMPACT_CATEGORY:** SUPPORT_BURDEN (display truth)
**IMPACT_FLAVOR:** misleading — primary metric is wrong for the most common reading pattern
**FIX:** Introduce `reading_events(id, user_id, document, occurred_at, percentage)` append-only table; compute Active/Streak from distinct dates there.
**SEVERITY:** CRITICAL — cornerstone metric persistently wrong for the most common reading pattern.
**DISCOVERED VIA:** Lens C
**LABELS:** ["display-truth", "stats", "schema-gap"]

### Finding P2: Year-in-Books and reading stats show zero for Kobo-only users
**WHAT THE USER SEES:**
1. Kobo-only user has hundreds of `kobo_reading_states` marked Finished.
2. Year-in-Books card shows `0 books finished`, `0 longest streak`, `0 days reading`.
3. Reading Activity card says "No reading activity recorded yet. Connect KOReader via Settings → KOSync..."
4. User concludes the app is broken or that their Kobo data isn't tracked.

**ROOT CAUSE:** Two parallel state systems with no unification. `GetYearInBooks`/`GetReadingStats`/`GetReadingStreak` query only `reading_progress` (KOSync). `GetRecommendations`/`ListGroupMemberProgress` query only `kobo_reading_states`.

**EVIDENCE:**
- `internal/db/reading_progress.go:203-228` — only `reading_progress`.
- `internal/db/recommendations.go:25-30` — only `kobo_reading_states`.
- `frontend/src/components/Dashboard.svelte:271-278` — hard-coded KOSync-only copy.

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** misleading — primary stats wrong for entire user class
**FIX:** Unify reading-state writes (write to both tables) OR UNION ALL both in aggregate queries OR introduce a unified `reading_events` table.
**SEVERITY:** CRITICAL — primary stats persistently wrong for an entire user class (Kobo users).
**DISCOVERED VIA:** Lens C
**LABELS:** ["display-truth", "dual-system", "kobo"]

---

## HIGH

### Finding P3: AI tag generation is documented + shipped server-side but unreachable from the UI
**WHAT THE USER SEES:**
1. User opens a book; the "AI Enrichment Review" panel only appears when an enrichment already exists.
2. There's no "Generate AI tags" / "Fetch AI enrichment" button anywhere in the frontend.
3. The only way to trigger `/api/books/{id}/metadata/ai-fetch` is curl.

**ROOT CAUSE:** `fetchAIEnrichment` is exported in `frontend/src/lib/api/metadata.ts:48-55` but no Svelte component calls it. `AIEnrichmentPanel.svelte` only renders an existing pending enrichment and exposes Apply/Reject.

**EVIDENCE:**
- Backend wired: `internal/handlers/metadata_ai.go:71-100` (POST `/api/books/{id}/metadata/ai-fetch`).
- Job handler registered: `cmd/server/main.go:136`.
- API client: `frontend/src/lib/api/metadata.ts:48`. Only consumer is its own test file.
- `BookDetail.svelte:139` — mounts AIEnrichmentPanel; no fetch button.

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** dark feature — server, jobs, audit actions, Apply/Reject UI all exist for a flow nobody can start through the UI
**FIX:** Add a "Generate AI enrichment" button in `BookDetail.svelte` (or inside `AIEnrichmentPanel`) that calls `api.fetchAIEnrichment(bookId)`.
**SEVERITY:** HIGH — feature exists but cannot be triggered from supported entry point.
**DISCOVERED VIA:** Lens B
**LABELS:** ["feature-gap", "ui-missing"]

### Finding P4: "Currently Reading" list shows opaque KOReader hashes instead of book titles
**WHAT THE USER SEES:**
1. Dashboard → Reading Activity → Currently Reading.
2. Entries render like `0b3e72cb1c1c2bca9... 35%` — raw KOReader document MD5 with no human-readable title, author, cover, or link.

**ROOT CAUSE:** `reading_progress.document` is an opaque KOReader hash with no `book_id` column or hash→book mapping. `HandleReadingProgressStats` returns it raw; `Dashboard.svelte` renders it as-is.

**EVIDENCE:**
- `internal/handlers/reading_progress.go:96-104` — DTO carries `Document` unchanged.
- `internal/handlers/kosync.go:145-156,4` — document is the opaque KOReader hash.
- `frontend/src/components/Dashboard.svelte:317-353` — `{item.document}` shown directly.
- `db/migrations/sqlite/20260317000001_create_kosync_tables.sql:13-26` — no `book_id` column.

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** misleading — primary widget shows gibberish
**FIX:** Add `book_id` to `reading_progress` resolved by hashing book files on import/scan, then JOIN; OR at minimum render a friendlier label.
**SEVERITY:** HIGH — primary widget displays gibberish.
**DISCOVERED VIA:** Lens C
**LABELS:** ["display-truth", "schema-gap"]

### Finding P5: "Books Finished" migrates between years when a user re-opens a finished book
**WHAT THE USER SEES:**
1. User finished 5 books in 2025; Year-in-Books for 2025 shows `5 books finished`.
2. User re-opens one of those books in Jan 2026 (or KOReader's session-resume pings).
3. Year-in-Books for 2025 now shows `4 books finished`; 2026 shows `1 book finished`.
4. The book "moved" between years; historical record is non-durable.

**ROOT CAUSE:** `BooksFinished` counts rows by `updated_at IN year`. Since `(user_id, document)` is unique and `updated_at` is overwritten on every sync, opening a finished book moves its row to the new year.

**EVIDENCE:**
- `internal/db/reading_progress.go:203-207`.
- `internal/db/kosync.go:88-94` (upsert overwrites `updated_at`).

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** misleading — historical stat is non-stable
**FIX:** Add `first_finished_at` column (set once, never updated); bucket BooksFinished by that.
**SEVERITY:** HIGH — historical stat silently rewrites itself.
**DISCOVERED VIA:** Lens C
**LABELS:** ["display-truth", "history-rewrite"]

### Finding P6: Recommendations include books the user has already finished via KOSync
**WHAT THE USER SEES:**
1. KOReader user reads many books via KOSync; KOSync data lives in `reading_progress`.
2. Dashboard recommendations include books they already finished.
3. No way to dismiss persistently.

**ROOT CAUSE:** `user_reads` CTE in `recommendationsQuery` only excludes from `kobo_reading_states`, not from `reading_progress`.

**EVIDENCE:**
- `internal/db/recommendations.go:25-30,92` — exclusion only consults `kobo_reading_states`.

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** misleading — engine recommends already-read titles
**FIX:** UNION both reading-state sources in `user_reads` CTE (depends on resolving P4's mapping).
**SEVERITY:** HIGH — primary recommendation surface wrong for KOReader users.
**DISCOVERED VIA:** Lens C
**LABELS:** ["display-truth", "recommendations"]

### Finding P7: Libraries page shows "Add A Library" empty state when the API call fails
**WHAT THE USER SEES:**
1. User has 3 libraries.
2. `GET /api/libraries` returns 500 briefly.
3. Page renders the empty-state "Add A Library" button; sidebar shows no libraries.
4. User concludes their libraries were deleted; creates a duplicate. Scanner now indexes the same folders twice.

**ROOT CAUSE:** `LibraryStore.load()` silently swallows errors; no `loadError` field on the store. Consumers can't distinguish "no libraries yet" from "load failed".

**EVIDENCE:**
- `frontend/src/stores/libraries.svelte.ts:39-51` — `catch { /* silently fail */ }`.
- `frontend/src/components/Libraries.svelte:84-91` — shows empty-state button whenever `libraries.length === 0`.
- `frontend/src/components/Sidebar.svelte:36-39,195` — same issue.
- Contrast: `frontend/src/stores/groups.svelte.ts:17-23`, `frontend/src/stores/reading-lists.svelte.ts:17-20` correctly expose `loadError`.

**IMPACT_CATEGORY:** DATA_LOSS (user creates duplicate library state)
**IMPACT_FLAVOR:** misleading — user takes a destructive-ish action based on wrong UI signal
**FIX:** Mirror `readingListStore`/`groupStore` pattern: add `loadError`, set `loaded=true` on failure with error in distinct state; gate the empty-state UI on `loaded && !loadError`.
**SEVERITY:** HIGH — primary flow; user creates unwanted state because UI lied.
**DISCOVERED VIA:** Lens A / E
**LABELS:** ["stale-ui", "empty-vs-error", "stores"]

### Finding P8: Settings tabs swallow load errors, presenting empty forms with "Configured" badges — admin overwrites stored config
**WHAT THE USER SEES:**
1. Admin opens Settings → SMTP (or OIDC/Watch Folder/LLM).
2. `getSmtpConfig` transient 500.
3. Tab renders with empty fields; status pill at top still reads "Configured".
4. Admin retypes the form and saves — overwrites the previously saved config.

**ROOT CAUSE:** Multiple settings tabs catch the initial-load error with empty `catch {}` blocks; no banner, no disabled-save guard.

**EVIDENCE:**
- `frontend/src/components/settings/SmtpTab.svelte:66-85` — IIFE with `catch {}`.
- `frontend/src/components/Settings.svelte:82-104` — same for OIDC/getConfigStatus.
- `frontend/src/components/settings/WatchFolderTab.svelte:28-42`.
- `frontend/src/components/settings/LLMTab.svelte:47-65`.
- `frontend/src/components/settings/CalibreImportTab.svelte:45-53` (libraries-list).
- Backend exposes the trap: `internal/handlers/config_smtp.go:124-127` reuses existing password only if username unchanged AND existing password non-empty — admin changing the username with the (silently absent) password gets 400 "password is required".

**IMPACT_CATEGORY:** DATA_LOSS
**IMPACT_FLAVOR:** blocking — admin can't update config without losing stored data; can lock themselves out of OIDC
**FIX:** Surface load errors as a banner and disable Save until the read succeeds. Single fix template applies to all four tabs.
**SEVERITY:** HIGH — admin-only but high-cost recovery; can lock the instance out of working SMTP/OIDC.
**DISCOVERED VIA:** Lens A / E
**LABELS:** ["settings", "silent-failure", "data-overwrite"]

### Finding P9: Group member progress hides KOReader readers from co-readers
**WHAT THE USER SEES:**
1. Reading group: one Kobo user, one KOReader user.
2. Group progress page for a shared book shows the Kobo user's progress.
3. The KOReader user appears as "0%, never updated" even when they're 70% through.

**ROOT CAUSE:** `ListGroupMemberProgress` joins only `kobo_reading_states`; no `reading_progress` source.

**EVIDENCE:**
- `internal/db/reading_groups.go:312-323` — single LEFT JOIN against `kobo_reading_states`.

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** misleading — co-readers can't see each other's actual progress
**FIX:** Same union-or-mapping work as P2/P4/P6.
**SEVERITY:** HIGH — feature broken for mixed-device groups (a primary use case).
**DISCOVERED VIA:** Lens C
**LABELS:** ["display-truth", "groups", "dual-system"]

### Finding P10: Delete-book leaves files orphaned on disk; admin's "clean up" doesn't reclaim disk
**WHAT THE USER SEES:**
1. Admin deletes 50 books to clean up.
2. `du -sh /library` shows zero change.
3. The DB cascades books → book_files rows; but the actual EPUB/PDF/MOBI files on disk remain forever.

**ROOT CAUSE:** Delete handlers only call DB `DELETE`. No `os.Remove` for the disk file. No periodic GC job.

**EVIDENCE:**
- `internal/db/books.go:252-255` — only DB delete.
- `internal/handlers/book_files.go:133-141` — uses `deleteResource` with no disk cleanup.
- `grep -rn "os.Remove" internal/handlers/book*.go` — only staging-failure paths.

**IMPACT_CATEGORY:** SUPPORT_BURDEN (operability)
**IMPACT_FLAVOR:** blocking — admin can't clean up disk via the UI
**FIX:** Two-part: (a) capture file paths during delete, validate under library root, `os.Remove` post-commit. (b) Add periodic GC job for `.uploads/` and orphaned files.
**SEVERITY:** HIGH — primary admin flow promises cleanup but doesn't deliver.
**DISCOVERED VIA:** Lens D
**LABELS:** ["data-cleanup", "broken-promise"]

---

## MEDIUM

### Finding P11: Dashboard shows "Total Books: 0" instead of placeholder on count-endpoint failure
**WHAT THE USER SEES:**
1. User has 487 books.
2. `GET /api/stats/books/count` returns 500.
3. Dashboard shows error banner AND "Total Books: 0" — contradictory.

**ROOT CAUSE:** `Dashboard.svelte:56` sets `totalBooks = 0` on error; the `stats` derived only treats `null` as "loading".

**EVIDENCE:**
- `frontend/src/components/Dashboard.svelte:49-58,110-114,224-241`.

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** misleading — error state shown as plausible value
**FIX:** Drop the `totalBooks = 0` line; let `null` render as "…" or show "—" when `countError` is set.
**SEVERITY:** MEDIUM — wrong information on the most prominent landing screen.
**DISCOVERED VIA:** Lens A / E
**LABELS:** ["display-truth", "error-state"]

### Finding P12: Passkey section silently disappears if `getPasskeyEnabled()` fails — user can't manage credentials
**WHAT THE USER SEES:**
1. User has a registered passkey.
2. `GET /api/auth/passkeys/enabled` returns 500.
3. Settings → Account hides the entire Passkeys section because `passkeyEnabled` defaults to false.
4. User can't see/revoke their existing passkey, but it still works on the login screen.

**ROOT CAUSE:** `PasskeysSection.svelte:33-42` catches the error silently; section is wrapped in `{#if passkeyEnabled}` with default `false`.

**EVIDENCE:**
- `frontend/src/components/settings/PasskeysSection.svelte:19,33-42,105`.

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** confusing — feature appears disabled when it's actually working
**FIX:** Track the error and render the section with an error state; separate "feature enabled" from "list credentials".
**SEVERITY:** MEDIUM — account/auth management primary flow degraded.
**DISCOVERED VIA:** Lens A
**LABELS:** ["silent-failure", "credentials"]

### Finding P13: "Metadata Fetch already in progress…" UI branch is unreachable; double-click creates duplicate pending rows
**WHAT THE USER SEES:**
1. User clicks "Fetch Metadata" twice rapidly.
2. Both POSTs return `enqueued`; the frontend never enters the "already_running" branch.
3. Two `goodreads_metadata` pending rows for the same `(user_id, book_id)`.
4. User applies/rejects the latest; the older one resurfaces next visit. "Why does this keep coming back?"

**ROOT CAUSE:** Backend defines `metadataStatusAlreadyRunning` constant but no handler returns it. `enqueueEnrichmentJob` lacks `WithUnique`. No DB partial unique on `(user_id, book_id) WHERE status='pending'`.

**EVIDENCE:**
- `internal/handlers/metadata_dto.go:60-64` — constant declared, never used.
- `internal/handlers/metadata.go:158` — `Enqueue` without `WithUnique`.
- Contrast: `internal/handlers/book_upload.go:197` DOES use `WithUnique`.

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** confusing — pending rows zombie back
**FIX:** Add `jobs.WithUnique(5*time.Minute)` to the Enqueue, and convert `asynq.ErrTaskIDConflict` into `{"status":"already_running"}`. Apply same fix to `enrich:ai` path. Optional: partial unique index.
**SEVERITY:** MEDIUM — secondary flow with reappearing-state UX problem.
**DISCOVERED VIA:** Lens B
**LABELS:** ["dedup", "ghost-rows"]

### Finding P14: SSE error event tells user "failed" even when retry succeeds; pending banner only appears on navigate-away-and-back
**WHAT THE USER SEES:**
1. "Fetch Metadata" → transient failure → SSE publishes `EventError`.
2. UI shows "failed to fetch book" and clears spinner.
3. asynq retries; retry succeeds; complete event has no subscribers.
4. User stares at "fetch failed" but the system has fresh pending metadata.
5. Pending banner only shows after navigate-away-and-back.

**ROOT CAUSE:** Job publishes terminal SSE events on every attempt; frontend treats first `error` as final.

**EVIDENCE:**
- `internal/jobs/enrich_goodreads.go:90-96,119-126` — publishes EventError + returns error → retry.
- `internal/worker/worker.go:223` — default MaxRetry=5.
- `internal/handlers/metadata_sse.go:121-124` — closes connection on `error`.
- `frontend/src/components/books/MetadataFetchPanel.svelte:162-166` — closes SSE; doesn't `loadPendingMetadata`.

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** misleading — "failed" displayed while success exists
**FIX:** On EventError, frontend calls `loadPendingMetadata()` to recover state. Or only publish EventError when asynq exhausts retries (on archive).
**SEVERITY:** MEDIUM — primary metadata flow shows wrong outcome.
**DISCOVERED VIA:** Lens B
**LABELS:** ["sse", "retry", "stale-ui"]

### Finding P15: groupStore / readingListStore mark `loaded=true` on failure — retry requires page reload
**WHAT THE USER SEES:**
1. `/api/groups` (or `/api/reading-lists`) fails once transiently.
2. Error banner shown; no retry button.
3. Switching tabs and coming back doesn't refetch.

**ROOT CAUSE:** Both stores set `this.loaded = true` in the catch block.

**EVIDENCE:**
- `frontend/src/stores/groups.svelte.ts:14-26` (line 24 sets loaded=true in catch).
- `frontend/src/stores/reading-lists.svelte.ts:10-23` (same).
- Contrast: `frontend/src/stores/crudStore.svelte.ts:20-33` (correct).

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** blocking (workaround = full reload)
**FIX:** Remove `this.loaded = true` from catch in both stores; or add retry buttons mirroring `Tags.svelte:145-153`.
**SEVERITY:** MEDIUM — primary navigation surface degraded.
**DISCOVERED VIA:** Lens E
**LABELS:** ["stores", "retry"]

### Finding P16: SMTP load failure → admin retypes; backend rejects "password is required" because UI hid existing-password state
**WHAT THE USER SEES:**
1. Admin opens Settings → SMTP. `GetSmtpConfig` returns 500.
2. Form is blank but status pill says "Configured". Password placeholder reads "Enter your SMTP password" (fresh-install copy) because `smtpStatus.passwordSet` defaulted to false.
3. Admin enters new host without re-entering password.
4. Backend rejects: "password is required when username is set".
5. Admin can't update SMTP without re-entering the password they never saw.

**ROOT CAUSE:** Cousin of P8. `SmtpTab.svelte:66-85` silently catches load failure; `smtpStatus.passwordSet` stays at the initial-state `false`; the password reuse logic only kicks in when username unchanged AND existing password non-empty.

**EVIDENCE:**
- `frontend/src/components/settings/SmtpTab.svelte:44-54,66-85,293-308`.
- `internal/handlers/config_smtp.go:99-127` (password reuse logic).

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** blocking — admin stuck on a config flow
**FIX:** Same as P8 — surface the load error and gate Save.
**SEVERITY:** MEDIUM — admin-only stuck state with workaround (re-enter password).
**DISCOVERED VIA:** Lens A
**LABELS:** ["settings", "silent-failure"]

### Finding P17: No max-length cap on entity names (authors, series, tags, libraries, reading lists, groups)
**WHAT THE USER SEES:**
1. User pastes a 50 KB clipboard into "Create Tag".
2. Form submits with 201; tag created.
3. The tag breaks the autocomplete dropdown — a 50 KB blob in every paginated list response.

**ROOT CAUSE:** `validateName` only checks non-empty; `normalizeName` only trims+collapses whitespace. Columns are `TEXT NOT NULL` with no length CHECK.

**EVIDENCE:**
- `internal/handlers/validate.go:26-32`.
- `internal/db/authors.go:22-28`.
- `db/migrations/sqlite/20260313020000_create_authors_table.sql:4` and equivalents.
- Contrast: `internal/handlers/validate.go:38` (token names DO have 100-char cap).

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** blocking — UI degradation hard to recover
**FIX:** Extend `validateName` with a max-length cap (256 bytes). Add DB CHECK constraint.
**SEVERITY:** MEDIUM — easy footgun across many endpoints.
**DISCOVERED VIA:** Lens D
**LABELS:** ["validation", "max-length"]

### Finding P18: Libraries can be created with overlapping or identical paths — every book double-counted
**WHAT THE USER SEES:**
1. Admin creates Library A with path `/books`.
2. Later creates Library B with path `/books/fiction`.
3. Scanner runs both. Every fiction file is processed twice. Library counts double-count those titles.

**ROOT CAUSE:** `validateAndPrepareLibrary` checks each path exists/is directory but never cross-checks against existing libraries.

**EVIDENCE:**
- `internal/handlers/libraries.go:119-170` — no overlap check.
- `db/migrations/sqlite/20260313010000_remove_user_id_from_libraries.sql:4` — UNIQUE on name only.

**IMPACT_CATEGORY:** DATA_LOSS (duplicate data)
**IMPACT_FLAVOR:** misleading — admin gets double-counts with no warning
**FIX:** In `validateAndPrepareLibrary`, compare each incoming path against existing libraries' paths; reject equal/subpath/superpath with 409 listing the offending pair.
**SEVERITY:** MEDIUM — admin-only but high-pain to clean up.
**DISCOVERED VIA:** Lens D
**LABELS:** ["validation", "overlap"]

### Finding P19: POST /api/groups/{id}/members silently no-ops for non-owners (returns 204 success)
**WHAT THE USER SEES:**
1. Non-owner member tries to add a friend to the group.
2. Backend returns 204 No Content with no error.
3. DB does nothing (SQL filters by owner_id).
4. Member sees "added" UX but the friend never appears.

**ROOT CAUSE:** `AddGroupMember` returns `(added bool, err error)`. The handler only logs audit when `added` is true and unconditionally writes 204. Non-owner case is indistinguishable from already-member case.

**EVIDENCE:**
- `internal/handlers/group_members.go:77-95`.
- `internal/db/reading_groups.go` (owner_id filter in SQL).

**IMPACT_CATEGORY:** SUPPORT_BURDEN (silent failure)
**IMPACT_FLAVOR:** misleading — success shown on failed action
**FIX:** Distinguish "already-member" (idempotent 204) from "not owner" (return `ErrNotGroupOwner` → 403).
**SEVERITY:** MEDIUM — primary collaboration flow silently fails.
**DISCOVERED VIA:** Lens D
**LABELS:** ["silent-failure", "groups"]

### Finding P20: POST /api/book-files/{id}/email has no per-user rate limit; account-takeover → SMTP fan-out abuse
**WHAT THE USER SEES:** A user (or stolen API key) loops POSTs to email different recipients. No per-user limit. SMTP provider eventually blocks the operator's account.

**ROOT CAUSE:** `/api/book-files/` is wrapped only in `requireAuth`. No specific limiter on `/email`. Compare `/api/config/smtp/test` which IS limited.

**EVIDENCE:**
- `internal/server/routes.go:103,62`.

**IMPACT_CATEGORY:** BRAND_DAMAGE
**IMPACT_FLAVOR:** abuse — admin's mail reputation damaged
**FIX:** Wrap `/api/book-files/.../email` in a per-user rate limiter (e.g., N emails/hour).
**SEVERITY:** MEDIUM — feature-completes-Phase-3-Finding-6 (which covered the open-relay path).
**DISCOVERED VIA:** Lens D
**LABELS:** ["rate-limit", "abuse"]

### Finding P21: Signup email not normalized (no lowercase) — same human creates duplicate accounts
**WHAT THE USER SEES:**
1. User signs up with `Alice@Example.com`.
2. Later forgets and signs up with `alice@example.com`. Both succeed.
3. She has two accounts; can't remember which casing for login.

**ROOT CAUSE:** goauth `Signup` only trims; doesn't lowercase. Project wrapper doesn't normalize either.

**EVIDENCE:**
- `goauth@v0.6.0/handler/auth.go:112-117` — only trim.
- `internal/handlers/auth_compat.go:217-249` — no custom normalization.

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** confusing — duplicate accounts
**FIX:** Lowercase email in the project's Signup wrapper before delegating to goauth. Same for Login.
**SEVERITY:** MEDIUM — onboarding fragility.
**DISCOVERED VIA:** Lens D
**LABELS:** ["normalization", "signup"]

### Finding P22: Signup accepts malformed emails (no @ etc.); downstream email features fail silently
**WHAT THE USER SEES:** User signs up with `"my email"` (no @). 201. Later, send-to-device or admin SMTP test silently fails because RFC5322 parsing rejects it. User never knows why "email this book" doesn't work.

**ROOT CAUSE:** goauth `Signup` doesn't `mail.ParseAddress`; only emptiness check. Other email surfaces DO validate.

**EVIDENCE:**
- `goauth@v0.6.0/handler/auth.go:112-117`.
- `internal/handlers/book_file_email.go:64-68` (strict parse here).
- `internal/handlers/config_smtp.go:233-241` (strict here too).

**IMPACT_CATEGORY:** SUPPORT_BURDEN
**IMPACT_FLAVOR:** misleading — feature appears broken later
**FIX:** Wrap goauth Signup; call `mail.ParseAddress` first; 400 "invalid email" upfront.
**SEVERITY:** MEDIUM — onboarding fragility pairs with P21.
**DISCOVERED VIA:** Lens D
**LABELS:** ["validation", "signup"]

---

## LOW

### Finding P23: ReadingListDetail shows blank page during initial fetch (no loading state)
**WHAT THE USER SEES:** User bookmarks `#reading-lists/abc` and opens in a new tab. During the initial load, the page body is blank — no spinner, no message — until the store resolves.
**ROOT CAUSE:** `ReadingListDetail.svelte:104-262` has `{#if error}` and `{#if list}` but no `{:else if !loaded}` branch.
**EVIDENCE:** `frontend/src/components/reading-lists/ReadingListDetail.svelte:38-46,115-119`. Contrast `BookDetail.svelte:95-105`.
**FIX:** Add `{:else if !readingListStore.loaded}` loading branch.
**SEVERITY:** LOW — cosmetic but appears broken.
**DISCOVERED VIA:** Lens A
**LABELS:** ["loading-state"]

### Finding P24: Whitespace-only book titles accepted on create/update
**ROOT CAUSE:** `createBook`/`updateBook` check `req.Title == ""` literally — no `strings.TrimSpace`.
**EVIDENCE:** `internal/handlers/book_crud.go:92-95,204-207`. Contrast `internal/handlers/validate.go:27`.
**FIX:** Use `strings.TrimSpace(req.Title) == ""` and assign trimmed value back.
**SEVERITY:** LOW — book unfindable by title.
**DISCOVERED VIA:** Lens D
**LABELS:** ["validation", "trim"]

### Finding P25: POST /books/{id}/files accepts negative size and unsupported file_type
**ROOT CAUSE:** `postBookFiles` validates only non-empty; doesn't call `LookupSupportedFileType` or check `FileSize > 0`; doesn't verify book exists.
**EVIDENCE:** `internal/handlers/books_files.go:53-72`. Contrast `book_upload.go:129-133`.
**FIX:** Add type lookup, positive-size check, and pre-`GetBook` 404.
**SEVERITY:** LOW — API consumers only.
**DISCOVERED VIA:** Lens D
**LABELS:** ["validation", "api"]

### Finding P26: PUT /api/groups/{id} returns 404 to non-owner members instead of 403
**ROOT CAUSE:** SQL filters by `owner_id`; member-but-not-owner gets `sql.ErrNoRows` → 404.
**EVIDENCE:** `internal/db/reading_groups.go:144-157`, `internal/handlers/named_entity.go:275-277`.
**FIX:** Split into membership/ownership checks; return 403 when member-but-not-owner.
**SEVERITY:** LOW — confusing error code only.
**DISCOVERED VIA:** Lens D
**LABELS:** ["http-status", "groups"]

### Finding P27: Library path edit doesn't trigger fresh scan; 24h dedup blocks fix
**WHAT THE USER SEES:** Admin notices the library path is wrong, edits to correct path, saves — no new scan runs.
**ROOT CAUSE:** `createLibrary` enqueues with `WithUnique(24h)`. `updateLibrary` doesn't enqueue at all. Even if it did, dedup key wouldn't reflect the path change.
**EVIDENCE:** `internal/handlers/libraries.go:220-232,278-312`.
**FIX:** In `updateLibrary`, enqueue a fresh `scan:library` job; compute unique key from `library_id + paths_hash`. Optional: add "Scan now" button.
**SEVERITY:** LOW — admin must wait or restart; not actively broken, just stuck.
**DISCOVERED VIA:** Lens D
**LABELS:** ["scan", "dedup", "admin"]
