# Action Trace Findings

## Phase Metadata
- Actions traced: 107 (10 parallel sub-agents covering all action categories)
- Actions with bugs found: 30+ across all surfaces
- Raw findings before consolidation: 38
- Findings reported here (pre-validation): 28
- Validation in Phase 5 will confirm or disprove each one.

---

## CRITICAL

### Finding 1: First-user-admin race grants admin to a second concurrent signup
**WHAT HAPPENS:**
1. Operator stands up a fresh Biblioteka instance (no `INITIAL_ADMIN_*` env vars; common default).
2. Attacker who can reach the public listener races the operator's first signup with their own.
3. Both signup requests pass their respective `SELECT EXISTS(email=...)` checks.
4. Both then read `SELECT COUNT(*) FROM users` — both see 0 because neither INSERT has committed.
5. Both INSERTs commit with `is_admin = TRUE`.
6. Attacker now has full admin authority over a server they don't own.

**IMPACT_CATEGORY:** SECURITY_BREACH
**BLAST RADIUS:** Every fresh Biblioteka install that does not pre-seed admin via `INITIAL_ADMIN_*` env. Window is small on SQLite (WAL serializes writers but not the read-then-write pair), seconds wide on PostgreSQL.
**VERIFICATION ARTIFACT:**
- `internal/db/users.go:53-73` — `CreateUser` issues three separate `QueryRowContext` calls (EXISTS, COUNT, INSERT) without `BeginTx`, advisory lock, or `INSERT…WHERE NOT EXISTS`.
- `db/migrations/sqlite/20260224000000_add_is_admin_to_users.sql:1-2` — no unique partial index `(is_admin) WHERE is_admin = TRUE`.
- `internal/db/users.go:79-99` — `CreateOIDCUser` is structurally identical; same race via OIDC callback.
- `internal/server/bootstrap.go:19-71` — `seedInitialAdmin` only fires when env vars set.
**WHY:** "First user is admin" is encoded as a read-then-write in app code across two transactions, not a single atomic SQL statement or schema constraint.
**FIX:** Use a single SQL statement: `INSERT INTO users (..., is_admin) VALUES (..., NOT EXISTS (SELECT 1 FROM users)) RETURNING ...`. Or wrap the count+insert in a SERIALIZABLE transaction.
**SEVERITY:** CRITICAL — passes the test: a real attacker on a real default config gains admin on every fresh install.
**DISCOVERED VIA:** Auth/signup trace
**LABELS:** ["auth-race", "first-user-admin", "signup"]

### Finding 2: Any authenticated user can DELETE any book, wiping it for every user on the instance
**WHAT HAPPENS:**
1. Alice (any logged-in user, including via stolen API key) calls `DELETE /api/books/{id}` for any book in the shared library.
2. Handler dispatches to `deleteResource(...)` — no admin gate; only `requireAuth` middleware on the route.
3. The book row is deleted; ON DELETE CASCADE removes `book_files`, `book_authors`, `book_series`, `book_tags`, `library_books`, `book_annotations`, `book_downloads`, `kobo_reading_states`, `reading_list_books`.
4. EVERY other user on the instance loses (a) the ability to find/download the book, (b) their annotations on the book, (c) their reading progress for the book, (d) their download history.
5. The file on disk is left orphaned (see Finding 11).
**IMPACT_CATEGORY:** DATA_LOSS
**BLAST RADIUS:** Every multi-user instance. Any authenticated user can wipe shared library data for every other user.
**VERIFICATION ARTIFACT:**
- Route: `internal/server/routes.go:97` — `s.mux.Handle("/api/books/", s.requireAuth(...))` (no `requireAdmin`).
- Handler: `internal/handlers/book_crud.go:261` — `deleteResource(...)` with no admin gate.
- Cascade migrations: `db/migrations/sqlite/20260413000001_create_reading_list_books_table.sql:4`, `20260414000004_create_book_annotations_table.sql:5`, `20260317000002_create_kobo_reading_states_table.sql:5`, `20260412180727_create_book_downloads_table.sql:4-5`.
- Contrast: `internal/handlers/libraries.go:189,279,329` — library create/update/delete all DO call `requireAdmin(h.DB, w, r)`.
**WHY:** Books are intentionally shared across users (no `user_id` column), but the DELETE handler was never adjusted to require admin authorization to match the "shared resource" model.
**FIX:** Wrap `deleteBook` in `if !requireAdmin(h.DB, w, r) { return }` (matching the library pattern). Same fix required for the related deletes in Finding 3.
**SEVERITY:** CRITICAL — DATA_LOSS that is unrecoverable for other users' annotations/reading progress; firable by any authenticated user.
**DISCOVERED VIA:** Book CRUD trace
**LABELS:** ["authz", "shared-resource", "data-loss"]

### Finding 3: Any authenticated user can DELETE any book_file / author / series / tag, wiping shared metadata for everyone
**WHAT HAPPENS:**
1. Alice calls `DELETE /api/tags/{id}` for any tag (or `/api/authors/{id}`, `/api/series/{id}`, `/api/book-files/{id}`).
2. No admin gate; only `requireAuth`.
3. The row is deleted; cascade removes `book_tags` / `book_authors` / `book_series` rows for EVERY user.
4. Books that were tagged become invisible in tag-filtered views. Books whose only author was that record become orphaned. Book-file rows can be wiped for files that other users were downloading.
**IMPACT_CATEGORY:** DATA_LOSS (and SUPPORT_BURDEN)
**BLAST RADIUS:** Every multi-user instance.
**VERIFICATION ARTIFACT:**
- `internal/handlers/book_files.go:133` — `deleteBookFile` — no `requireAdmin`.
- `internal/handlers/authors.go:222` — `deleteAuthor` — no `requireAdmin`.
- `internal/handlers/series.go:219` — `deleteSeries` — no `requireAdmin`.
- `internal/handlers/tags.go:160` — `deleteTag` — no `requireAdmin`.
- Route registrations `internal/server/routes.go:75-84,103` — all `s.requireAuth` only.
**WHY:** Same root cause as Finding 2 — shared resources should require admin.
**FIX:** Add `requireAdmin` to each of the four delete handlers. (Could be a single PR with Finding 2.)
**SEVERITY:** CRITICAL — same severity rationale as Finding 2; consolidating with it is reasonable in implementation but the affected entities are distinct.
**DISCOVERED VIA:** Book CRUD trace
**LABELS:** ["authz", "shared-resource", "data-loss"]

### Finding 4: ExifTool argument injection via newline in uploaded filename
**WHAT HAPPENS:**
1. Attacker uploads a book file with a POSIX-legal filename containing `\n` (Linux ext4 allows newlines in filenames), e.g. `foo\n-overwrite_original.epub`.
2. `HandleUpload` sanitizes only via `filepath.Base()` which strips path separators but not newlines.
3. The staged path `<libRoot>/.uploads/abc123_foo\n-overwrite_original.epub` is sent to ExifTool's stay-open protocol via `fmt.Fprintln(e.stdin, file)`.
4. ExifTool's stay-open protocol treats each newline-separated stdin line as a separate argument, so the path splits into two args; `-overwrite_original` is a real ExifTool flag that triggers in-place metadata writes.
5. Once the stay-open protocol desyncs, the bufio scanner waits for a `{ready}` token in the wrong position, the extractor's `markDead()` fires permanently, and ALL future metadata extraction silently fails (see Finding 5).
**IMPACT_CATEGORY:** PROD_INCIDENT (paired with permanent metadata extractor death)
**BLAST RADIUS:** Any user with upload permission (every authenticated user). Single malicious upload kills the global metadata pipeline.
**VERIFICATION ARTIFACT:**
- `internal/handlers/book_upload.go:128` — `filename := filepath.Base(header.Filename)` only strips `/`.
- `internal/jobs/scan_directory.go:119-131` — `filepath.Abs(path)` likewise doesn't strip `\n`; scanned files with newlines have same injection path.
- `internal/exif/exif.go:250` — `fmt.Fprintln(e.stdin, file)` writes one line per filename; multiline filename → multiple stdin lines.
- ExifTool docs: `-overwrite_original`, `-config`, `-tagsfromfile` are all real flags.
**WHY:** Filename sanitization assumes only path separators are dangerous, but newlines control the ExifTool stay-open protocol framing.
**FIX:** In `book_upload.go:128` and `scan_directory.go:119`, reject filenames containing `\n`, `\r`, or any control char. Or, in `internal/exif/exif.go:250`, URL-encode the path before writing, with a corresponding decode in any wrapper that reads the path back (ExifTool itself accepts `-charset filename=utf8`).
**SEVERITY:** CRITICAL — exploitable by any authenticated user, persistent denial of a primary feature (metadata extraction) plus possible file-mutation side effects via ExifTool's in-place flags.
**DISCOVERED VIA:** Book upload + process:file pipeline trace
**LABELS:** ["filename-injection", "exiftool", "argv-split"]

### Finding 5: One malformed input file permanently disables metadata extraction (no restart logic)
**WHAT HAPPENS:**
1. A user uploads (or a scan picks up) any file that causes ExifTool to emit > 64KB of TSV output for a single record (large embedded thumbnails, verbose metadata, etc.) OR a file that crashes ExifTool, OR triggers Finding 4.
2. `bufio.Scanner` default 64KB max → `ErrBufferTooSmall` (or other I/O error).
3. `Extractor.markDead()` fires; `e.dead = true` for the lifetime of the process.
4. EVERY subsequent `ExtractMetadataFromFile` returns `ErrDead`.
5. All future ingests silently fall back to filename-only metadata; SSE clients see no progress events; users get books with no titles/authors/covers.
6. No restart logic anywhere in `metadata.Extractor` — server must be redeployed to recover.
**IMPACT_CATEGORY:** PROD_INCIDENT
**BLAST RADIUS:** Every instance, on first malformed file (intentional or accidental).
**VERIFICATION ARTIFACT:**
- `internal/metadata/extractor.go:32-39` — `NewExiftool(ctx)` called with no options; no buffer override.
- `internal/exif/exif.go:122-126` — `bufferSet` false by default; default 64KB scanner.
- `internal/exif/exif.go:267-291` — any scanner error/EOF calls `markDead()`.
- `internal/exif/exif.go:218-219` — `if e.dead { return nil, ErrDead }` blocks all future calls.
- `cmd/server/main.go:99-103` — single extractor passed to `NewProcessFileHandler`; no respawn.
**WHY:** Single-process subprocess pattern with no health-check / restart wrapper.
**FIX:** (a) Pass `WithBuffer(maxScanTokenSize)` setting a much larger scanner buffer (say 4 MiB). (b) Add a Restart() method to `*exif.Exiftool` that respawns the subprocess on death; the metadata Extractor calls it on next request after `ErrDead`.
**SEVERITY:** CRITICAL — denial of metadata extraction for the entire instance, triggered by any user upload.
**DISCOVERED VIA:** Book upload + process:file pipeline trace
**LABELS:** ["pipeline-death", "subprocess", "no-restart"]

### Finding 6: Open SMTP relay — any authenticated user can email any book to any external address
**WHAT HAPPENS:**
1. Alice (any authenticated user) POSTs `/api/book-files/{id}/email` with `{"to":"victim@anywhere.com"}`.
2. Handler validates only RFC5322 syntax — no per-user allowlist, no rate limit, no domain restriction, no check that the recipient is associated with Alice.
3. The server attaches the book file (up to 25 MB) and sends via its configured SMTP relay using the operator's IP reputation.
4. Alice can repeat indefinitely (no per-user quota; the auth limiter is only on `/api/auth/*`).
5. Operator's SMTP relay is now a spam/distribution channel; their IP can be reputation-flagged; legitimate mail starts bouncing.
**IMPACT_CATEGORY:** BRAND_DAMAGE (and PROD_INCIDENT for the mail server)
**BLAST RADIUS:** Every instance with SMTP enabled and more than one user (including any instance with public signup enabled).
**VERIFICATION ARTIFACT:**
- Handler: `internal/handlers/book_file_email.go:44-176` — only `mail.ParseAddress(to)` + 25 MB size cap; no rate limit, no allowlist, no per-user limit.
- Route: `internal/server/routes.go:103` — `s.requireAuth(...)` (no `authLimiter.Wrap`).
- Compare: `internal/server/routes.go:26-27` — `/api/auth/login` is wrapped in `authLimiter`.
**WHY:** The send-to-device feature was modeled on a single-user assumption (the user emails THEIR book to THEIR device); the multi-user impact wasn't considered.
**FIX:** Require the `to` field to be a user-configured "device address" (stored in users table or per-user setting); reject arbitrary recipients. Add per-user rate limit (e.g., 5 emails/day).
**SEVERITY:** CRITICAL — easily abusable open relay on any multi-user instance; reputational/compliance damage to operator's mail infrastructure.
**DISCOVERED VIA:** Book CRUD trace
**LABELS:** ["open-relay", "abuse", "smtp"]

### Finding 7: OIDC discovery + JWKS fetches at runtime bypass SafeHTTPClient (DNS rebinding SSRF)
**WHAT HAPPENS:**
1. Admin saves OIDC issuer `https://attacker.com`. The handler's *validation* discovery correctly uses `ssrf.SafeHTTPClient`.
2. After validation passes, `OnOIDCConfigSet` runs and calls `handlers.NewOIDCHandler(ctx, ...)` with the raw request context (no `oidc.ClientContext`).
3. `NewOIDCHandler` → `oidc.NewProvider(ctx, issuerURL)` (in goauth, no SSRF wrapper) → `getClient(ctx)` falls through to `http.DefaultClient`.
4. The provider's `RemoteKeySet` captures this context for its LIFETIME via `context.WithoutCancel(ctx)`.
5. On every subsequent JWT validation, JWKS GETs go through the unprotected client. DNS for `attacker.com` resolves to `127.0.0.1` or `169.254.169.254`. Server connects to its own loopback / cloud metadata service; response is parsed as JWKS or surfaced in error messages.
**IMPACT_CATEGORY:** SECURITY_BREACH (SSRF to internal services / cloud metadata)
**BLAST RADIUS:** Every instance with OIDC enabled. Triggered any time JWKS is refreshed (typically per JWT validation cache miss).
**VERIFICATION ARTIFACT:**
- Safe path: `internal/handlers/config_oidc.go:162-171` — uses `oidc.ClientContext(safeCtx, ssrf.SafeHTTPClient(0))`.
- Unsafe path: `internal/handlers/config_oidc.go:205-206` → `internal/server/init_handlers.go:173-181` — passes raw `r.Context()` with no `oidc.ClientContext`.
- Downstream: `internal/handlers/auth_compat.go:69-70` (`goauthhandler.NewOIDCHandler(ctx, ...)`)
- goauth: `~/go/pkg/mod/github.com/amalgamated-tools/goauth@v0.6.1/handler/oidc.go:60` — no `oidc.ClientContext`.
- coreos: `~/go/pkg/mod/github.com/coreos/go-oidc/v3@v3.18.0/oidc/oidc.go:88` — defaults to `http.DefaultClient`.
- Persistent context: `~/go/pkg/mod/github.com/coreos/go-oidc/v3@v3.18.0/oidc/jwks.go:71` — `ctx: context.WithoutCancel(ctx)`.
**WHY:** Validation and runtime use different HTTP-client paths; only validation was wrapped with SSRF protection.
**FIX:** In `OnOIDCConfigSet`, wrap ctx with `oidc.ClientContext(ctx, ssrf.SafeHTTPClient(timeout))` before calling `NewOIDCHandler`. Or modify goauth to always install an SSRF-safe client.
**SEVERITY:** CRITICAL — SSRF to internal infra in default config, reachable via admin-configurable URL, persistent for the lifetime of the configured provider.
**DISCOVERED VIA:** Admin config + SSRF trace
**LABELS:** ["ssrf", "dns-rebinding", "oidc"]

### Finding 8: SMTP send bypasses SSRF protection via DNS rebinding
**WHAT HAPPENS:**
1. Admin sets `SMTP_HOST=attacker.com`. `smtp.ValidateHost` accepts any non-literal-private hostname; does NOT resolve DNS.
2. On send, `smtp.Send` uses raw `net.Dialer{Timeout: 10s}` which resolves at dial time.
3. Attacker-controlled DNS returns `127.0.0.1` (or `169.254.169.254`, or internal RFC1918).
4. Server connects to its own loopback / internal services and speaks SMTP commands. STARTTLS banner content is leaked to the admin via test-email error messages.
**IMPACT_CATEGORY:** SECURITY_BREACH
**BLAST RADIUS:** Every instance with SMTP enabled where an admin can configure host.
**VERIFICATION ARTIFACT:**
- Literal-only check: `internal/smtp/config.go:140-152` — `net.ParseIP` + `IsLoopbackHost` reject literal/`localhost`; no DNS resolution.
- Vulnerable dial: `internal/smtp/send.go:65` — raw `net.Dialer`, used by all three TLS modes at lines 70, 82, 97.
- Compare: `internal/ssrf/ssrf.go:67-101` — `SafeHTTPClient` re-resolves at connect time.
**WHY:** The recent `a3bd1640` fix added literal-IP/"localhost" rejection at config time but didn't address DNS rebinding at dial time.
**FIX:** Implement `ssrf.SafeDialer` that re-resolves and validates IP inside the SMTP dial step, mirroring `SafeHTTPClient`. Or at minimum resolve at `ValidateHost` time and reject all-private resolutions.
**SEVERITY:** CRITICAL — SSRF reachable in default config via admin-configurable URL; can probe loopback or cloud-metadata services.
**DISCOVERED VIA:** Admin config + SSRF trace
**LABELS:** ["ssrf", "dns-rebinding", "smtp"]

### Finding 9: MOBI/AZW3 cover decode OOM via crafted dimensions
**WHAT HAPPENS:**
1. Attacker uploads a MOBI/AZW3 with a `coverlength` pointing to a tiny PNG/GIF whose declared dimensions are 50000×50000 (or larger).
2. `GetMobiCover` calls `image.Decode(io.LimitReader(f, coverlength))`. Go's `image.Decode` has no built-in dimension cap; it allocates a pixel buffer for the declared dimensions regardless of input bytes.
3. ~10 GB allocation. OOM kill. Worker crashes. Restart loop.
**IMPACT_CATEGORY:** PROD_INCIDENT
**BLAST RADIUS:** Every instance accepting MOBI/AZW3 uploads.
**VERIFICATION ARTIFACT:**
- `internal/exif/mobi_cover.go:48` — `image.Decode` with no prior `image.DecodeConfig` size check.
- `internal/coverutil/decode.go` — enforces only encoded-bytes size (20 MB), not pixel count.
**WHY:** Cover-size guard is byte-based; doesn't account for decompressed pixel-buffer size.
**FIX:** Call `image.DecodeConfig(...)` first to get dimensions; reject if `width*height > maxPixelBudget` (e.g., 50 megapixels).
**SEVERITY:** CRITICAL — single malicious upload crashes the worker; reachable by any authenticated user with upload permission.
**DISCOVERED VIA:** Book upload + process:file pipeline trace
**LABELS:** ["decoder-bomb", "oom", "image-decode"]

---

## HIGH

### Finding 10: Orphan file with no DB record after failed createBookRecord (asynq retries skip the file)
**WHAT HAPPENS:**
1. `process:file` moves a staged upload to its reorganized location.
2. `CreateBookWithFile` then fails (transient DB error, FK violation, timeout).
3. asynq retries the job. `resolveSourcePath` does `os.Stat(p.Path)` on the original staged path — fails because file was moved.
4. Falls back to `reorganizedCandidatePaths` derived from `pathparser.ParseBookPath(p.Path, libraryRoot)`, but the original path is `<libRoot>/.uploads/<hex>_file.epub`, so the parsed candidate doesn't match the real moved destination.
5. Returns "source file no longer exists and could not be located, skipping" — every retry skips silently.
6. End state: real file at `<libRoot>/Author/Title/file.epub` with NO `books`/`book_files`/sidecars/library link.
7. Only recovered if (a) library is monitored AND (b) `scan:libraries` 24h cron walks it.
**IMPACT_CATEGORY:** DATA_LOSS
**BLAST RADIUS:** Every instance during transient DB errors; permanent orphan for non-monitored libraries.
**VERIFICATION ARTIFACT:**
- `internal/jobs/book_record_helpers.go:117-200` — `createBookRecord` no compensating move-back on DB failure.
- `internal/jobs/book_path_helpers.go:118-145` — recovery candidates derived from staging path.
- `internal/jobs/process_book_file.go:109-120` — reorganize happens BEFORE createBookRecord.
- `internal/organize/organize.go:353-367` — `sanitizeDirName` strips leading dot; `.uploads` → `uploads`.
**WHY:** Reorganize is not atomic with DB write; recovery logic doesn't know the post-move path.
**FIX:** On DB write failure, move the file back to its staging path before propagating the error (compensating action). Or persist the destination path in the asynq payload after move so retries know where to look.
**SEVERITY:** HIGH — silent orphan file accumulation; data-loss class.
**DISCOVERED VIA:** Book upload + process:file pipeline trace
**LABELS:** ["orphan-file", "compensation", "saga"]

### Finding 11: Unbounded staged upload accumulation (disk-fill DoS)
**WHAT HAPPENS:**
1. User (malicious or legitimate) uploads a 500 MB file (the maxUploadSize) that fails extraction (corrupt EPUB, etc.).
2. `process:file` retries 5 times, then archives the task. No cleanup of the staged file in `<libRoot>/.uploads/` is ever performed.
3. The watch-folder cron (every 1m) keeps detecting the file but the 24h dedup prevents re-enqueue. After 24h elapses, another 5 retries fire and again leave the file.
4. Disk fills. Server eventually crashes.
**IMPACT_CATEGORY:** PROD_INCIDENT
**BLAST RADIUS:** Every instance; reachable by any authenticated user.
**VERIFICATION ARTIFACT:**
- `internal/handlers/book_upload.go:33` — `uploadStagingDir = ".uploads"`.
- `internal/handlers/book_upload.go:197-212` — cleanup only on enqueue error.
- `grep -rn "\.uploads" cmd/ internal/` — zero references to cleanup logic.
- `cmd/server/main.go:138-146` — only `scan:libraries` and `scan:watch-folder` cron jobs; no janitor.
**WHY:** No janitor was implemented for the staging directory; failure paths were not connected to cleanup.
**FIX:** Add a scheduled `janitor:uploads` job that removes files in `.uploads/` older than 24h. Also remove the staged file in the `process:file` failure path (asynq's `OnFailure` hook).
**SEVERITY:** HIGH — disk-fill DoS triggerable by any authenticated user.
**DISCOVERED VIA:** Book upload + process:file pipeline trace
**LABELS:** ["disk-fill", "orphan-cleanup", "dos"]

### Finding 12: scan:libraries walks .uploads/, causing duplicate-row race with in-flight uploads
**WHAT HAPPENS:**
1. A user uploads a file; the staged file sits in `<libRoot>/.uploads/...` waiting for `process:file`.
2. `scan:libraries` → `scan:library` → `scan:path` does `filepath.WalkDir` from the library root with no skip for `.uploads/`.
3. Scanner enqueues a SECOND `process:file` task for the staged path with a DIFFERENT payload shape (`UserID:""`, no metadata overrides) — asynq dedup doesn't catch it because the payload hash differs.
4. Two workers race on the same staged file: one wins the rename, the other's rename fails. The loser proceeds to `createBookRecord` with the original staging path, but the file has already been moved. DB ends up with two `books` rows + two `book_files` rows, one of which points to a deleted file.
**IMPACT_CATEGORY:** DATA_LOSS (broken downloads) + SUPPORT_BURDEN (duplicate books)
**BLAST RADIUS:** Every instance with at least one library; race window scales with upload size and scan frequency.
**VERIFICATION ARTIFACT:**
- `internal/jobs/scan_directory.go:82-147` — `filepath.WalkDir` recurses unconditionally; no skip for `.uploads`.
- `internal/jobs/scan_libraries.go:40-48` — enqueues `scan:library` for library paths.
- `internal/handlers/book_upload.go:145` — staging dir is always inside library root.
- `internal/jobs/book_record_helpers.go:21-105` — rename-fail fallback uses original path without fresh duplicate check.
**WHY:** Staging directory wasn't excluded from library scans; deduplication keys differ between upload and scan code paths.
**FIX:** Add `if d.IsDir() && d.Name() == ".uploads" { return fs.SkipDir }` in `ScanDirectory`. Optional: normalize the asynq payload for `process:file` so dedup catches the race.
**SEVERITY:** HIGH — guaranteed corruption under load; silent.
**DISCOVERED VIA:** Book upload + process:file pipeline trace
**LABELS:** ["scan-race", "duplicate-rows", "process-file"]

### Finding 13: OIDC `registration_disabled` gate locks out existing OIDC users
**WHAT HAPPENS:**
1. Admin sets `registration_disabled=true` (intent: "no new users").
2. Admin's session expires; admin signs in via OIDC.
3. `OIDCHandler.Callback` wrapper sees `registration_disabled=true` → 403 "signup is disabled" — short-circuits BEFORE delegating to goauth (which would have found the existing user via `FindByOIDCSubject` and issued a JWT).
4. Admin is now locked out via OIDC. Existing OIDC-only users (`password_hash=''`) cannot fall back to password login (goauth rejects empty hashes).
5. If admin was the only admin and uses OIDC exclusively, recovery requires direct DB edits.
**IMPACT_CATEGORY:** SUPPORT_BURDEN (admin lockout)
**BLAST RADIUS:** Every OIDC-enabled instance that ever flips `registration_disabled`.
**VERIFICATION ARTIFACT:**
- Gate: `internal/handlers/auth_compat.go:80-92`.
- Lockout cause: `goauth@v0.6.1/handler/auth.go:170-174` (Login rejects empty `password_hash`).
- OIDC user lookup short-circuited: `goauth@v0.6.1/handler/oauth2_common.go:107-118` (`findOrCreateUser` tries `FindByOIDCSubject` first).
- Test confirms behavior: `auth_compat_test.go:211-223` (`TestOIDCHandler_Callback_BlockedByRegistrationDisabled`) does not test existing-user case.
**WHY:** The wrapper blocks all OIDC callbacks rather than only the `CreateOIDCUser` branch.
**FIX:** In the wrapper, gate only the CREATE path: decode the id_token, call `FindByOIDCSubject(sub)`, and only return 403 when the user does NOT exist.
**SEVERITY:** HIGH — admin lockout on a documented admin action; recovery requires DB surgery.
**DISCOVERED VIA:** Auth/signup trace
**LABELS:** ["oidc", "registration-gate", "lockout"]

### Finding 14: OIDC first-time signup is not audit-logged
**WHAT HAPPENS:**
1. New user signs in via OIDC for the first time → `findOrCreateUser` → `CreateOIDCUser` inserts a new row in `users`.
2. The biblioteka wrapper at `auth_compat.go:80-92` does NOT capture the response or emit an audit log.
3. Direct signup via `POST /api/auth/signup` IS audited (`auth_compat.go:228-248`).
4. Result: a new user account materializes with zero audit-trail entry. Combined with Finding 1's race, an OIDC first-user-admin grant would also be invisible.
**IMPACT_CATEGORY:** SUPPORT_BURDEN (audit gap)
**BLAST RADIUS:** Every OIDC-enabled instance; every first-time OIDC login.
**VERIFICATION ARTIFACT:**
- Direct-signup audit (works): `internal/handlers/auth_compat.go:228-248`.
- OIDC callback wrapper (missing audit): `internal/handlers/auth_compat.go:80-92`.
- User insertion site: `goauth@v0.6.1/handler/oauth2_common.go:119`.
**WHY:** Wrapper was built only for the registration gate; audit mirror was missed.
**FIX:** After successful callback, parse the id_token's `sub`, look up via `FindByOIDCSubject`, and emit `AuditActionUserSignedUp` when `created_at` is within a recent window.
**SEVERITY:** HIGH — audit emission gap on a sensitive auth action.
**DISCOVERED VIA:** Auth/signup trace
**LABELS:** ["audit", "oidc"]

### Finding 15: Raw Kobo token leaks into OTel trace span names
**WHAT HAPPENS:**
1. Kobo device sends `GET /kobo/<64-hex-token>/v1/library/sync`.
2. `TraceMiddleware` (chain position 2, before auth) sets the span name to the raw URL path, which CONTAINS the token.
3. Spans are exported to the OTel sink (Jaeger/Tempo/Honeycomb). Span names are indexed, low-cardinality-sensitive fields; trace UIs put them in dashboards and search.
4. Anyone with read access to traces can replay the token indefinitely against the Kobo endpoints — read+write the user's reading state, download books, etc.
**IMPACT_CATEGORY:** SECURITY_BREACH
**BLAST RADIUS:** Every instance with OTel tracing enabled; every Kobo-device request creates one exposed span.
**VERIFICATION ARTIFACT:**
- `internal/otel/tracing.go:24-26` — span name = `r.Method + " " + r.URL.Path` (raw URL).
- `internal/server/server.go:220-226` — middleware chain: Trace runs BEFORE the Kobo middleware path rewrite.
- `internal/auth/kobo_middleware.go:51-99` — path rewrite happens later.
**WHY:** Trace middleware uses raw URL; route-template extraction is not implemented.
**FIX:** In `TraceMiddleware`, for paths under `/kobo/`, set span name to `<method> /kobo/{token}/...` (templated). Apply same change to LoggingMiddleware (Finding 16).
**SEVERITY:** HIGH — long-lived bearer credential exfiltrated to a routine observability backend.
**DISCOVERED VIA:** KOSync/Kobo trace
**LABELS:** ["token-leak", "observability", "kobo"]

### Finding 16: Raw Kobo token leaks into DEBUG access logs
**WHAT HAPPENS:** Same shape as Finding 15 but via LoggingMiddleware at DEBUG level. Lower default exposure but logs are commonly shipped to long-retention aggregators (Loki/Datadog/ELK).
**IMPACT_CATEGORY:** SECURITY_BREACH
**BLAST RADIUS:** Any instance whose operator ever turns DEBUG logging on (common when troubleshooting Kobo sync).
**VERIFICATION ARTIFACT:**
- `internal/handlers/middleware/logging.go:73, 87` — `slog.String(otelkeys.URL, r.URL.String())`.
- `internal/server/server.go:223` — `LoggingMiddleware` before mux.
**FIX:** Same as Finding 15.
**SEVERITY:** HIGH — same class as 15 with smaller default surface.
**DISCOVERED VIA:** KOSync/Kobo trace
**LABELS:** ["token-leak", "logging", "kobo"]

### Finding 17: `logAudit` failure warning omits userID/action/entityType/entityID — audit holes are unreconstructible
**WHAT HAPPENS:**
1. A sensitive action (signup, password change, credential upsert, API key create, passkey delete, etc.) succeeds.
2. `logAudit` calls `db.CreateAuditLog`, which fails for a transient reason.
3. `logAudit` emits a warn-level slog event with ONLY `slog.Any(otelkeys.Error, err)` — no userID, no action, no entityType, no entityID.
4. The user-visible action returns success. The audit log row was never written. There is no way to reconstruct what was lost from the logs.
**IMPACT_CATEGORY:** SUPPORT_BURDEN / compliance
**BLAST RADIUS:** Every sensitive flow on every instance during any transient DB failure.
**VERIFICATION ARTIFACT:**
- `internal/handlers/dberrors.go:99-103` — `logAudit` warning body.
- Contrast: `internal/handlers/crud.go:174-183` — `deleteResourceCore` audit warning attaches resource/entityType/idKey but still misses userID/action.
- Affected call sites: `internal/handlers/named_entity.go:89,144,228,285`, `credentials.go:213,256`, `tokens_compat.go:65`, `book_subresource.go:70`, `auth_compat.go:116,129,156,170,243,257`.
**WHY:** The "audit must not fail the request" policy was implemented without preserving the fallback evidence the policy depends on.
**FIX:** In `logAudit`, include all audit fields in the WarnContext call (userID, action, entityType, entityID, marshaled metadata).
**SEVERITY:** HIGH — structural blind spot affecting every sensitive flow.
**DISCOVERED VIA:** Shared DB helpers trace
**LABELS:** ["audit", "observability"]

### Finding 18: Sidecar `cover.jpg.tmp` overwrite race corrupts covers under default organization
**WHAT HAPPENS:**
1. Two `process:file` jobs writing the same directory (which happens by default under `book_per_folder` and `none` organization types where `baseName==""`).
2. Both target `<dir>/cover.jpg` and the SAME non-unique temp `<dir>/cover.jpg.tmp`.
3. Goroutine A: `os.WriteFile("cover.jpg.tmp", bytesA)` succeeds.
4. Goroutine B: `os.WriteFile("cover.jpg.tmp", bytesB)` overwrites (truncate).
5. Goroutine A: `os.Rename("cover.jpg.tmp", "cover.jpg")` succeeds — but file holds bytesB.
6. Two books in the same directory end up with the wrong cover; OPF manifest references stale image.
**IMPACT_CATEGORY:** DATA_LOSS (corruption)
**BLAST RADIUS:** Every instance with concurrent uploads/scans into the same directory.
**VERIFICATION ARTIFACT:**
- `internal/sidecar/cover.go:43-49` — non-unique `tmp := path + ".tmp"`.
- Compare `internal/sidecar/opf.go:199-220` — uses `os.CreateTemp(dir, opfName+".tmp-*")` (unique).
- `internal/sidecar/naming.go:23-44` — for `none` org default, `baseName==""`.
**FIX:** Switch to `os.CreateTemp(dir, "cover.jpg.tmp-*")` to get a unique temp name. Last-rename-wins is still possible but no longer corrupts mid-write.
**SEVERITY:** HIGH — silent data corruption affecting cover display.
**DISCOVERED VIA:** Book upload + process:file pipeline trace
**LABELS:** ["sidecar", "race", "concurrent-write"]

### Finding 19: PUT /api/books/{id} has no optimistic locking — concurrent edits silently overwrite
**WHAT HAPPENS:**
1. Two users edit the same book simultaneously.
2. Both fetch the current book, edit different fields, PUT.
3. No `If-Match` ETag, no version column, no row-version check.
4. Last writer wins; earlier writer sees 200 OK but their changes are gone.
5. The "Apply Goodreads metadata" flow has the same issue between its `GetBook` and `UpdateBook` calls.
**IMPACT_CATEGORY:** DATA_LOSS (silent lost-update)
**BLAST RADIUS:** Every multi-user instance.
**VERIFICATION ARTIFACT:**
- `internal/handlers/book_crud.go:198-246` — `updateBook`: no version check, no transaction.
- `internal/db/books.go:236-249` — `UpdateBook`: SET all fields unconditionally.
- `internal/handlers/metadata_goodreads.go:107-164` — `applyMetadata`: separate `GetBook` then `UpdateBook` calls.
**FIX:** Add a `version` column to `books`; PUT requires `If-Match` header carrying the version; SQL UPDATE includes `AND version = $N`.
**SEVERITY:** HIGH — lost-update affecting a frequent flow on a shared resource.
**DISCOVERED VIA:** Book CRUD trace
**LABELS:** ["lost-update", "concurrency"]

### Finding 20: parseLimitOffset allows offset up to 200,000 with no rate limit on list endpoints
**WHAT HAPPENS:**
1. Authenticated user (or stolen API key) sends `GET /api/books?offset=200000&limit=200` repeatedly.
2. Each request forces the DB to scan/skip 200,000 rows.
3. List endpoints have no `authLimiter` (only `/api/auth/*` does).
4. DB CPU usage climbs; legitimate requests queue.
**IMPACT_CATEGORY:** PROD_INCIDENT
**BLAST RADIUS:** Every instance with authenticated users.
**VERIFICATION ARTIFACT:**
- `internal/handlers/pagination.go:11,40-55` — `maxPageOffset = maxPageLimit * 1000 = 200_000`.
- `internal/server/routes.go:87-100` — list endpoints lack `authLimiter`.
**FIX:** Cap `maxPageOffset` at e.g. 10,000 or switch to cursor-based pagination on the largest tables. Optional: apply a general per-user rate limiter to list endpoints.
**SEVERITY:** HIGH — authenticated DoS amplification.
**DISCOVERED VIA:** Shared DB helpers trace
**LABELS:** ["pagination", "dos", "rate-limit"]

### Finding 21: SSE write-deadline is silently a no-op due to missing Unwrap on statusRecorder
**WHAT HAPPENS:**
1. SSE handler `streamEvents` calls `http.NewResponseController(w).SetWriteDeadline(...)` to extend per-write deadline beyond the server's 120 s WriteTimeout.
2. The response-writer chain is `gzipResponseWriter → statusRecorder → http.response`. `statusRecorder` does NOT implement `Unwrap()`, so `ResponseController` cannot reach the underlying http.response that supports `SetWriteDeadline`.
3. `SetWriteDeadline` returns `http.ErrNotSupported`; handler logs a warning and proceeds.
4. Every SSE connection is killed at 120 s (server's hard WriteTimeout). Long metadata-fetch flows lose progress events; UI hangs.
**IMPACT_CATEGORY:** SUPPORT_BURDEN
**BLAST RADIUS:** Every instance; every metadata SSE stream longer than 120 s.
**VERIFICATION ARTIFACT:**
- `internal/handlers/middleware/logging.go:14-50` — `statusRecorder` has no `Unwrap()`.
- `internal/handlers/middleware/gzip.go:71-73` — `gzipResponseWriter.Unwrap()` correct.
- `internal/handlers/metadata_sse.go:55-60` — handler calls SetWriteDeadline.
**FIX:** Add `func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }` to logging.go.
**SEVERITY:** HIGH — primary SSE feature degraded; silent (warning only).
**DISCOVERED VIA:** Middleware trace
**LABELS:** ["sse", "responsecontroller", "unwrap"]

### Finding 22: LoggingMiddleware always logs `user_id=""` on the "Request completed" line
**WHAT HAPPENS:**
1. `LoggingMiddleware` runs BEFORE auth (middleware chain order in `server.go`).
2. It calls `auth.UserIDFromContext(r.Context())` at the start — userID is always empty at that point.
3. It then calls `next.ServeHTTP(rec, r)` (where auth runs and mutates context downstream).
4. After completion, it logs `userID` using the captured empty-string variable.
5. EVERY access log entry shows `user_id=""`. Forensic queries like "which user just hit /api/admin/users/" return useless results.
**IMPACT_CATEGORY:** SUPPORT_BURDEN (forensic gap)
**BLAST RADIUS:** Every request on every instance.
**VERIFICATION ARTIFACT:**
- `internal/handlers/middleware/logging.go:65-93` — userID captured at line 68 before `ServeHTTP`.
- `internal/server/server.go:220-226` — LoggingMiddleware outside the mux that contains auth.
- Contrast: audit-log path correctly reads userID AFTER auth (e.g. `admin.go:107,155`).
**FIX:** At the completion log, re-read `auth.UserIDFromContext(rec.Request.Context())` — or move LoggingMiddleware INSIDE the mux per-route, after auth. Both work; the first is less invasive.
**SEVERITY:** HIGH — audit/forensics gap on every request.
**DISCOVERED VIA:** Middleware trace
**LABELS:** ["logging", "context", "audit"]

### Finding 23: No panic-recovery middleware — panicking handler leaves no slog entry / audit / gzip flush
**WHAT HAPPENS:**
1. A nil-pointer or out-of-bounds in any handler panics.
2. `net/http`'s default per-conn recover catches the panic (server doesn't crash) but: LoggingMiddleware's "Request completed" log never runs, GzipMiddleware's deferred `Close()` never runs (resource leak), audit log not emitted, client gets a closed connection with no body.
3. Operator sees "silent 502" with no slog entry to triage.
**IMPACT_CATEGORY:** PROD_INCIDENT (operability)
**BLAST RADIUS:** Every panicking handler.
**VERIFICATION ARTIFACT:**
- `internal/server/server.go:220-226` — middleware chain has no recover middleware.
- `grep recover() internal/server/ internal/handlers/middleware/ internal/auth/` — zero hits.
**FIX:** Add a `Recover` middleware at the outermost position that: calls `recover()`, logs the stack, increments a counter, writes a 500 response.
**SEVERITY:** HIGH — operability gap; turns every future handler bug into a silent-disappear mystery.
**DISCOVERED VIA:** Middleware trace
**LABELS:** ["panic-recovery", "operability"]

### Finding 24: POST /api/books/{id}/files — any user can register unlimited book_file rows pointing to any library file
**WHAT HAPPENS:**
1. Alice POSTs `{"file_path": "<libRoot>/path/to/any/book.epub"}` to `/api/books/{id}/files`.
2. Handler validates only that the path is under a library root.
3. No UNIQUE constraint on `(book_id, file_path)`. No check that the file isn't already registered. No ownership of the book.
4. Alice repeats N times, inflating storage stats, polluting search, and attaching arbitrary library files to attacker-created book rows.
**IMPACT_CATEGORY:** DATA_LOSS (data pollution)
**BLAST RADIUS:** Every multi-user instance.
**VERIFICATION ARTIFACT:**
- `internal/handlers/books_files.go:53-85` — no duplicate check, no ownership verification.
- `db/migrations/sqlite/20260313080000_create_book_files_table.sql:2-12` — no UNIQUE on `(book_id, file_path)`.
**FIX:** Add UNIQUE constraint on `(book_id, file_path)`. Optional: add admin gate (same shared-resource argument as Finding 2/3).
**SEVERITY:** HIGH — silent data pollution by any authenticated user.
**DISCOVERED VIA:** Book CRUD trace
**LABELS:** ["authz", "data-pollution"]

---

## MEDIUM

### Finding 25: GetReadingList does not honor group sharing — shared lists are inaccessible to recipients
**WHAT HAPPENS:**
1. User A shares list L with group G.
2. User B (member of G) calls `GET /api/groups/G/lists` and sees L.
3. B clicks through to `GET /api/reading-lists/L` → 404.
4. No `/api/groups/G/lists/L` endpoint exists either. No way for B to view the books in the shared list.
**IMPACT_CATEGORY:** SUPPORT_BURDEN
**BLAST RADIUS:** Every group using list sharing.
**VERIFICATION ARTIFACT:**
- `internal/db/reading_lists.go:64-74` — `GetReadingList` `WHERE rl.id=$1 AND rl.user_id=$2` (owner-only).
- `internal/db/reading_lists.go:127-140` — `verifyReadingListOwnership` returns `sql.ErrNoRows` when caller isn't owner.
- No alternate endpoint exists (handler grep).
- Test confirms owner-only behavior with no group-share test: `internal/db/reading_lists_test.go:106-117`.
**FIX:** Update `GetReadingList` (and `ListReadingListBooks`) to include `OR EXISTS (SELECT 1 FROM reading_group_lists rgl JOIN reading_group_members rgm ON rgm.group_id=rgl.group_id WHERE rgl.reading_list_id=rl.id AND rgm.user_id=$2)`.
**SEVERITY:** MEDIUM — feature half-built; not a security bug.
**DISCOVERED VIA:** Groups + annotations trace
**LABELS:** ["feature-gap", "groups"]

### Finding 26: ListGroupMemberProgress shares reading progress with no opt-in
**WHAT HAPPENS:**
1. User joins any group.
2. Every other member can call `GET /api/groups/{id}/progress?book_id=X` and see this user's `percent_read` and `updated_at` for ANY book.
3. There is no "share progress with group" toggle.
**IMPACT_CATEGORY:** SUPPORT_BURDEN (privacy expectation gap)
**BLAST RADIUS:** Every group member.
**VERIFICATION ARTIFACT:**
- `internal/db/reading_groups.go:299-325` — `ListGroupMemberProgress` LEFT JOINs `kobo_reading_states` for every member without any consent flag.
- No opt-in column in `reading_group_members`: see `db/migrations/sqlite/20260414000002_create_reading_group_members_table.sql:3`.
**FIX:** Add a `share_progress BOOLEAN DEFAULT FALSE` column to `reading_group_members`; filter on it in `ListGroupMemberProgress`. Or define this as intentional book-club behavior and document it.
**SEVERITY:** MEDIUM — may be intended product design; call out for product decision.
**DISCOVERED VIA:** Groups + annotations trace
**LABELS:** ["privacy", "groups"]

### Finding 27: Two admins concurrently demoting each other can leave zero admins (instance unrecoverable)
**WHAT HAPPENS:**
1. `HandleSetAdmin` blocks self-demotion (`targetID == callerID`).
2. Two admins A and B run concurrent PUTs (A demotes B; B demotes A).
3. Both pass the self-demotion check; both proceed.
4. `SetAdmin` is a plain UPDATE; no count-of-admins check.
5. Instance ends with 0 admins. Recovery requires direct DB access.
**IMPACT_CATEGORY:** SUPPORT_BURDEN
**BLAST RADIUS:** Every instance with ≥ 2 admins. Rare but recovery is high-cost.
**VERIFICATION ARTIFACT:**
- `internal/handlers/admin.go:163-166` — self-demotion guard.
- `internal/db/users.go:151-167` — `SetAdmin` plain UPDATE.
- No `CountAdmins` analogue in `users.go`.
**FIX:** In `SetAdmin`, wrap in tx; before demoting, check `SELECT COUNT(*) FROM users WHERE is_admin=TRUE AND id != $1 >= 1`.
**SEVERITY:** MEDIUM — rare but recovery is DB surgery.
**DISCOVERED VIA:** Admin config + SSRF trace
**LABELS:** ["admin-zero", "race"]

### Finding 28: `enrich:ai` enqueue lacks dedup on user-triggered path — future paid-LLM cost runaway
**WHAT HAPPENS:**
1. User clicks "Fetch AI metadata" twice rapidly (or refresh-spams).
2. `enqueueEnrichmentJob` does a pre-check for "pending row" but the check races the next enqueue.
3. Multiple jobs enqueued for the same (user, book). Each runs `provider.Generate` (LLM call).
4. asynq default `MaxRetry=5` multiplies the cost on a single transient failure.
5. Today (Ollama): CPU waste. Future (any paid LLM): unbounded spend per user with no quota.
**IMPACT_CATEGORY:** PROD_INCIDENT (forward-looking REVENUE_LEAK if provider becomes paid)
**BLAST RADIUS:** Every instance; cost depends on provider.
**VERIFICATION ARTIFACT:**
- `internal/handlers/metadata.go:158` — `Enqueue(ctx, jobType, payload)` with no `WithUnique`.
- Auto path uses `WithUnique(24*time.Hour)`: `internal/jobs/process_book_file.go:173` (for goodreads, not AI).
- `internal/worker/worker.go:215-228` — default MaxRetry=5.
**FIX:** Pass `WithUnique(60*time.Second)` on user-triggered AI enqueue; consider `WithMaxRetry(1)` for paid-API calls.
**SEVERITY:** MEDIUM — CPU waste today; budget-burn surface if provider changes.
**DISCOVERED VIA:** Goodreads/AI/recommendations trace
**LABELS:** ["dedup", "cost-control", "llm"]

---

## Negative results / verified clean (selected)

These angles were checked and intentionally not flagged:

- **Annotation private-leak**: `GetAnnotation` correctly enforces `(ba.user_id = $2 OR (ba.group_id IS NOT NULL AND ba.group_id IN (memberships)))` — private annotations are NOT exposed to group members.
- **Annotation write authorization**: only the author can update/delete; tests cover this.
- **Group annotation smuggling**: `checkOptionalGroupMembership` blocks attaching to a non-member group.
- **KOSync progress user-scoping**: `(user_id, document)` upsert key — two users with the same document MD5 do not collide.
- **Kobo reading state user-scoping**: queries filter by `(user_id, book_id)`.
- **OPDS download path traversal**: re-validated via `isBookFilePathAllowed` (symlink-aware).
- **OPDS XML escaping**: feeds use `encoding/xml`.
- **Asynqmon admin gate**: wrapped with `requireAdmin` at the routes mount.
- **LLM endpoint SSRF**: both validation-time `ssrf.ValidateURL` and runtime `ssrf.SafeHTTPClient` applied.
- **OIDC client secret encryption**: AES-GCM via goauth, key zeroized after init.
- **Recommendations user-scoping**: every CTE filtered by `user_id=$1`.
- **CSRF**: cookie `SameSite=Strict` confirmed via goauth.
- **JWT secret startup validation**: rejects too-short secrets before `ListenAndServe`.
- **Rate-limiter X-Forwarded-For trust**: requires `TRUSTED_PROXIES` env to honor.
- **`net/http.ServeMux` (Go 1.22+) pattern matching**: routes are exact-vs-prefix correctly assigned; no method-routing bypass.
- **Audit logs endpoint**: gated by `requireAdmin` inside the handler.
- **`ai_enrichments.RawResponse`**: `json:"-"` — never serialized.
- **Telemetry payload**: only `install_id`, `version`, `os`, `arch`; opt-in.
- **Goodreads SSRF**: endpoint hardcoded today; risk is forward-looking if it ever becomes configurable.
