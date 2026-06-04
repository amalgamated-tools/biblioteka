# Deep Review Findings

> Reviewed 2026-06-03. Commit: `0bbd70ec8bfec201e0449830d458e14a0f7518f5`. Repo: `amalgamated-tools/biblioteka`.
> Run via the Farfield Deep Review skill — [farfield.dev](https://farfield.dev)

## Summary

| # | Severity | Impact | Title | Location |
|---|---|---|---|---|
| 1 | HIGH | BRAND_DAMAGE | Open SMTP relay via send-to-device | `internal/handlers/book_file_email.go:44–176` |
| 2 | HIGH | SECURITY_BREACH | OIDC JWKS DNS-rebinding SSRF | `internal/server/init_handlers.go:173–181` |
| 3 | HIGH | SECURITY_BREACH | Kobo bearer token leaks into OTel span names | `internal/otel/tracing.go:23–26` |
| 4 | HIGH | DATA_LOSS | Any auth'd user can DELETE any book (no admin gate) | `internal/handlers/book_crud.go:261` |
| 5 | HIGH | DATA_LOSS | Any auth'd user can DELETE any author/series/tag/book_file | `internal/handlers/{authors,series,tags,book_files}.go` |
| 6 | HIGH | PROD_INCIDENT | ExifTool argument injection via newline in uploaded filename | `internal/handlers/book_upload.go:128` + `internal/exif/exif.go:250` |
| 7 | HIGH | PROD_INCIDENT | One bad file permanently disables metadata extraction (no restart) | `internal/exif/exif.go:122–126,198–219` |
| 8 | HIGH | PROD_INCIDENT | MOBI/AZW3 cover decode OOM via crafted dimensions | `internal/exif/mobi_cover.go:48` |
| 9 | HIGH | SUPPORT_BURDEN | Reading streak / active days undercounts the most common usage pattern | `internal/db/reading_progress.go:217–228` |
| 10 | HIGH | SUPPORT_BURDEN | Year-in-Books shows zero for Kobo-only users | `internal/db/reading_progress.go:203–228` |
| 11 | HIGH | SUPPORT_BURDEN | "Currently Reading" shows opaque KOReader hashes instead of titles | `internal/handlers/reading_progress.go:96–104` |
| 12 | MEDIUM | SECURITY_BREACH | First-user-admin race: two concurrent signups both become admin | `internal/db/users.go:53–73` |
| 13 | MEDIUM | SECURITY_BREACH | SMTP send bypasses SSRF via DNS rebinding | `internal/smtp/send.go:65` |
| 14 | MEDIUM | SECURITY_BREACH | Kobo bearer token leaks into DEBUG access logs | `internal/handlers/middleware/logging.go:73,87` |
| 15 | MEDIUM | SUPPORT_BURDEN | Disabling registration locks out existing OIDC-only users | `internal/handlers/auth_compat.go:80–92` |
| 16 | MEDIUM | SUPPORT_BURDEN | Two admins concurrently demoting each other → zero admins | `internal/handlers/admin.go:163–166` + `internal/db/users.go:151–167` |
| 17 | MEDIUM | DATA_LOSS | Orphan file in library tree after DB write failure during ingest | `internal/jobs/process_book_file.go:109–120` |
| 18 | MEDIUM | DATA_LOSS | scan:libraries walks .uploads/ → duplicate book rows from in-flight upload | `internal/jobs/scan_directory.go:82` |
| 19 | MEDIUM | SUPPORT_BURDEN | Delete-book leaves files on disk; admin "cleanup" doesn't reclaim space | `internal/db/books.go:252–255` |
| 20 | MEDIUM | SUPPORT_BURDEN | "Books Finished" silently migrates between years on re-open | `internal/db/reading_progress.go:203–207` |
| 21 | MEDIUM | SUPPORT_BURDEN | Recommendations include books KOReader users already finished | `internal/db/recommendations.go:25–30` |
| 22 | MEDIUM | SUPPORT_BURDEN | Group member progress hides KOReader readers from co-readers | `internal/db/reading_groups.go:312–323` |
| 23 | MEDIUM | DATA_LOSS | Libraries page shows empty-state on API failure → admin creates duplicate | `frontend/src/stores/libraries.svelte.ts:39–51` |
| 24 | MEDIUM | SUPPORT_BURDEN | SSE error event signals failure while asynq retry succeeds | `internal/jobs/enrich_goodreads.go:90–96` + `MetadataFetchPanel.svelte:162–166` |
| 25 | MEDIUM | SUPPORT_BURDEN | Library path edit doesn't trigger fresh scan; 24h dedup blocks fix | `internal/handlers/libraries.go:278–312` |
| 26 | MEDIUM | SUPPORT_BURDEN | Reading-group progress sharing has no per-user opt-in | `internal/db/reading_groups.go:299–325` |
| 27 | MEDIUM | SUPPORT_BURDEN | SSE write-deadline silently no-op due to missing Unwrap on statusRecorder | `internal/handlers/middleware/logging.go:14–50` |

## Findings

### Finding 1: Any authenticated user can email any book to any external address — server becomes an open mail relay

- **Severity**: HIGH
- **Impact category**: BRAND_DAMAGE
- **Location**: `internal/handlers/book_file_email.go:44–176`; route `internal/server/routes.go:103`
- **Trigger condition**: Any authenticated user POSTs `/api/book-files/{id}/email` with `{"to":"victim@anywhere.com"}` for any book they can read.
- **Consequence**: The configured SMTP relay sends arbitrary email through the operator's IP. Repeated abuse gets the operator's mail domain RBL-listed; legitimate notifications stop delivering.
- **Verdict**: CONFIRMED (Phase 4 P20 marked DUPLICATE and rolled in)

#### What happens

1. Attacker creates an account (signup is enabled by default per `internal/server/init_handlers.go:66` unless `DISABLE_SIGNUP=true`) or uses any compromised credential / API key.
2. Attacker POSTs `/api/book-files/{id}/email` with `to` set to an arbitrary RFC5322 address.
3. The handler validates only that the address parses (`mail.ParseAddress`). There is no per-user device-address allowlist, no domain restriction, and no rate limit on this route.
4. Server sends the book file (up to 25 MB) to the attacker-chosen recipient using its SMTP credentials and IP reputation.
5. Attacker loops with N different recipients; the server runs at line rate until the operator notices.

#### Root cause

The send-to-device flow was designed for a single user emailing THEIR book to THEIR e-reader; the multi-user abuse case wasn't considered. The handler at `internal/handlers/book_file_email.go:44–176` validates recipient only as RFC5322 syntax. No "device address" allowlist is stored per user, and no `authLimiter` wraps the route in `internal/server/routes.go:103` (compare `/api/config/smtp/test` at routes.go:62 which IS wrapped).

#### Evidence

- Buggy path: `internal/handlers/book_file_email.go:64–69` — only `mail.ParseAddress(to)` + 25 MB size cap.
- No rate limit on route: `internal/server/routes.go:103` — `s.requireAuth(...)` only.
- Comparison: `internal/server/routes.go:62` — `/api/config/smtp/test` wraps `s.authLimiter`.
- No per-user device-address allowlist in `db/migrations/sqlite/20260214235631_create_users_table.sql`.

#### How to verify

1. Sign up a second user (or use any API key).
2. `curl -X POST -H "Cookie: biblioteka_token=..." -H "Content-Type: application/json" -d '{"to":"any@external.example"}' http://localhost:8080/api/book-files/<any-file-id>/email`
3. Repeat with different addresses; observe SMTP sends in the operator's mail-server logs with no per-user throttle.

#### Suggested fix

Require `to` to be one of a per-user-configured set of "device email addresses" (add `users.device_emails` or a separate table). Reject arbitrary recipients with 400. Add a per-user rate limit (e.g., 5 sends/day) reusing the `authLimiter` plumbing. Both should land in the same PR.

#### Confidence

- Assumption verified: signup enabled by default (`init_handlers.go:66`).
- Assumption verified: no allowlist exists (grep across `internal/handlers/`, `internal/db/`, migrations).
- Assumption verified: route has no rate limit (`routes.go:103`).
- Proof script result: N/A (requires live SMTP).

#### Gate evidence

- IMPACT_CATEGORY: BRAND_DAMAGE — operator's mail IP can be RBL-listed.
- IS_LIVE_SURFACE: route is registered with only `requireAuth`.
- NO_SCENARIO_MITIGATION: no allowlist, no rate limit, recipient fully attacker-controlled.
- CTO_TEST: yes — a spammer using a self-hosted instance ruins the operator's mail reputation.

#### Labels

abuse, smtp, open-relay, rate-limit, impact:brand_damage

---

### Finding 2: OIDC JWKS fetch uses an SSRF-unprotected HTTP client — DNS-rebinding can leak cloud metadata or forge ID tokens

- **Severity**: HIGH
- **Impact category**: SECURITY_BREACH
- **Location**: `internal/handlers/config_oidc.go:205–206` → `internal/server/init_handlers.go:173–181` → `internal/handlers/auth_compat.go:69–75` → `goauth@v0.6.1/handler/oidc.go:60`
- **Trigger condition**: Admin saves an OIDC issuer whose discovery URL passes the validation-time SSRF check, but whose JWKS DNS later rebinds to a private IP (169.254.169.254, RFC1918).
- **Consequence**: Every subsequent JWT signature verification fetches JWKS via `http.DefaultClient`. Attacker-controlled DNS can route JWKS fetches to internal infrastructure, exfiltrating cloud-metadata credentials or serving attacker-controlled signing keys to forge ID tokens for arbitrary accounts.
- **Verdict**: CONFIRMED

#### What happens

1. Admin configures issuer `https://attacker.example.com`. Discovery validation at `config_oidc.go:162–171` correctly uses `oidc.ClientContext(safeCtx, ssrf.SafeHTTPClient(0))` and passes.
2. After validation, `OnOIDCConfigSet` runs `handlers.NewOIDCHandler(ctx, ...)` with the raw `r.Context()` — no `oidc.ClientContext` wrapper.
3. `goauth.NewOIDCHandler` calls `oidc.NewProvider(ctx, issuerURL)` without `oidc.ClientContext`; coreos go-oidc defaults to `http.DefaultClient`.
4. The provider's `RemoteKeySet` captures this context for its lifetime via `context.WithoutCancel(ctx)` (`coreos/go-oidc/v3@v3.18.0/oidc/jwks.go:71`).
5. On every JWKS cache miss, GET goes through the unprotected client. Attacker's DNS for `jwks.attacker.example.com` resolves to `169.254.169.254`.
6. Server connects there. Attacker either ingests internal metadata responses or feeds attacker-controlled signing keys to forge ID tokens for any user.

#### Root cause

Two HTTP-client paths for the OIDC provider — validation-time is SSRF-wrapped, runtime is not. `OnOIDCConfigSet` (`internal/server/init_handlers.go:173–181`) never received the same `oidc.ClientContext(safeCtx, ssrf.SafeHTTPClient(...))` treatment as the validation discovery.

#### Evidence

- Buggy runtime path: `internal/server/init_handlers.go:173–181` — passes raw `ctx`.
- Wrapper: `internal/handlers/auth_compat.go:69–75` — passes `ctx` unchanged.
- goauth: `~/go/pkg/mod/github.com/amalgamated-tools/goauth@v0.6.1/handler/oidc.go:60` — no `oidc.ClientContext`.
- coreos default: `~/go/pkg/mod/github.com/coreos/go-oidc/v3@v3.18.0/oidc/oidc.go:87–93` — `http.DefaultClient`.
- Persistent ctx: `~/go/pkg/mod/github.com/coreos/go-oidc/v3@v3.18.0/oidc/jwks.go:71` — `context.WithoutCancel(ctx)`.
- Comparison (safe): `internal/handlers/config_oidc.go:162–171`.

#### How to verify

1. Run a DNS server returning a public IP for `jwks.attacker.example.com` on first query, then `127.0.0.1` / `169.254.169.254` after.
2. Set OIDC issuer to a domain whose discovery returns `"jwks_uri": "https://jwks.attacker.example.com/keys"`.
3. After OIDC discovery succeeds, trigger any JWT validation after JWKS cache expires.
4. Observe the second JWKS request hitting the rebound IP.

#### Suggested fix

Inside `OnOIDCConfigSet` at `internal/server/init_handlers.go:173–181`, wrap the context before calling `handlers.NewOIDCHandler`:

```go
ctx = oidc.ClientContext(ctx, ssrf.SafeHTTPClient(30*time.Second))
oidcHandler, err := handlers.NewOIDCHandler(ctx, ...)
```

Or push the fix upstream into goauth so every consumer benefits.

#### Confidence

- Verified: keyset uses ctx for the lifetime of the provider (`jwks.go:71`).
- Verified: coreos defaults to `http.DefaultClient` (`oidc.go:87–93`).
- Verified: validation wrapper does NOT cover runtime handler.
- Proof script result: SKIPPED (requires DNS-rebind setup).

#### Gate evidence

- IMPACT_CATEGORY: SECURITY_BREACH — SSRF + potential token forgery.
- IS_LIVE_SURFACE: OIDC login flow uses the constructed handler on every JWT validation.
- NO_SCENARIO_MITIGATION: runtime path lacks SSRF wrapper.
- CTO_TEST: yes — SSRF to internal infra or JWT forgery halts sprints.

#### Labels

ssrf, dns-rebinding, oidc, impact:security_breach

---

### Finding 3: Raw Kobo device tokens leak into OpenTelemetry trace span names

- **Severity**: HIGH
- **Impact category**: SECURITY_BREACH
- **Location**: `internal/otel/tracing.go:23–26`; chain `internal/server/server.go:220–226`; rewrite `internal/auth/kobo_middleware.go:51–99`
- **Trigger condition**: Any Kobo device request to `/kobo/{token}/v1/...` while OTel tracing is enabled.
- **Consequence**: Each Kobo sync produces a span whose name CONTAINS the device's plaintext, long-lived bearer token. Anyone with read access to the operator's tracing backend gets a credential that lets them impersonate the device indefinitely.
- **Verdict**: CONFIRMED

#### What happens

1. Kobo device hits `GET /kobo/<64-hex-token>/v1/library/sync`.
2. `TraceMiddleware` (global, chain position 2) sets span name to raw `r.URL.Path`, BEFORE the mux that delegates to `requireKoboAuth`.
3. Span exports to whatever OTel backend is configured. Span names are first-class indexed fields.
4. Anyone with trace-read access can replay `GET /kobo/<that-token>/v1/library/sync` to read the user's library or PUT `/v1/library/{uuid}/state` to write reading-state on their behalf.

#### Root cause

The trace middleware uses raw `r.URL.Path` before the Kobo middleware extracts and strips the token. The Kobo middleware rewrites the path at `kobo_middleware.go:93–94`, but that runs inside the mux — after `TraceMiddleware` captured the span. Kobo tokens are hashed at rest (`internal/db/kobo_tokens.go:50–56`) precisely because plaintext-at-rest is unacceptable; plaintext-in-telemetry undoes that protection.

#### Evidence

- Span name from raw path: `internal/otel/tracing.go:24–25`.
- Middleware order: `internal/server/server.go:220–226`.
- Path rewrite happens later: `internal/auth/kobo_middleware.go:93–94`.
- Hashed-at-rest comparison: `internal/db/kobo_tokens.go:50–56`.

#### How to verify

1. Configure any OTel exporter; run `docker run jaegertracing/all-in-one`.
2. Add a Kobo token; sync a device or `curl https://server/kobo/<token>/v1/library/sync`.
3. Open Jaeger UI; span name contains the token.

#### Suggested fix

In `internal/otel/tracing.go`, for paths matching `^/kobo/`, set span name to a route template like `GET /kobo/{token}/v1/library/sync` (replace the second segment with `{token}`). Apply same fix to `internal/handlers/middleware/logging.go` (Finding 14). A shared helper that returns a route-templated name is cleanest.

#### Confidence

- Verified: chain ordering and rewrite location.
- Verified: span names are routinely visible to trace readers.
- Proof script result: N/A (requires OTel collector).

#### Gate evidence

- IMPACT_CATEGORY: SECURITY_BREACH — bearer-token disclosure.
- IS_LIVE_SURFACE: every Kobo request creates a span.
- NO_SCENARIO_MITIGATION: no scrubbing, no allow-list, no path-template extraction.
- CTO_TEST: yes — credential leak to third-party tracing backends.

#### Labels

token-leak, observability, kobo, otel, impact:security_breach

---

### Finding 4: Any authenticated user can DELETE any book — destroys other users' annotations, reading progress, and download history

- **Severity**: HIGH
- **Impact category**: DATA_LOSS
- **Location**: `internal/handlers/book_crud.go:261`; route `internal/server/routes.go:97`
- **Trigger condition**: Any authenticated user (including via stolen API key) calls `DELETE /api/books/{id}`.
- **Consequence**: The book is deleted; ON DELETE CASCADE wipes `book_files`, `book_authors`, `book_series`, `book_tags`, `library_books`, `book_annotations`, `book_downloads`, `kobo_reading_states`, `reading_list_books` — including EVERY OTHER USER's annotations, reading progress, and download history for that book.
- **Verdict**: CONFIRMED

#### What happens

1. Alice has 50 annotations and 30% reading progress on Book B.
2. Bob (any authenticated user) calls `DELETE /api/books/{B}`.
3. Handler dispatches to `deleteResource(...)` — no admin gate.
4. Book row deleted; cascade removes Alice's annotations, reading progress, download history, plus everyone else's data for this book.
5. The file on disk is left orphaned (see Finding 19).

#### Root cause

Books are shared by design (no `user_id` on `books`), but the DELETE handler was never adjusted to require admin authorization to match the "shared resource" model. The library CRUD handlers correctly require admin (`internal/handlers/libraries.go:189,279,329`); books/authors/series/tags do not.

#### Evidence

- Route: `internal/server/routes.go:97` — `s.requireAuth(...)` only.
- Handler: `internal/handlers/book_crud.go:261` — `deleteResource(...)` with no admin gate.
- Cascade migrations: `db/migrations/sqlite/20260414000004_create_book_annotations_table.sql:5` (`ON DELETE CASCADE`), `20260317000002_create_kobo_reading_states_table.sql:5`, `20260412180727_create_book_downloads_table.sql:4–5`.
- Comparison: `internal/handlers/libraries.go:189,279,329` — library CRUD DOES call `requireAdmin`.

#### How to verify

1. Create two users via signup.
2. As user A, upload a book and add an annotation.
3. As user B, `curl -X DELETE -H "Cookie: biblioteka_token=B" http://localhost:8080/api/books/<id>`.
4. Observe HTTP 204 and verify A's annotation is gone.

#### Suggested fix

In `internal/handlers/book_crud.go:deleteBook`, add at the top:

```go
if !requireAdmin(h.DB, w, r) {
    return
}
```

Matches the library-handler pattern.

#### Confidence

- Verified: route middleware chain.
- Verified: cascade migrations.
- Verified: handler body has no admin check.
- Proof script result: N/A (requires running server).

#### Gate evidence

- IMPACT_CATEGORY: DATA_LOSS — unrecoverable destruction of multi-user data.
- IS_LIVE_SURFACE: route registered; standard CRUD endpoint.
- NO_SCENARIO_MITIGATION: handler bypasses every guard except authentication.
- CTO_TEST: yes — non-admin destroying shared catalog and other users' annotations is sprint-blocking.

#### Labels

authz, shared-resource, data-loss, impact:data_loss

---

### Finding 5: Any authenticated user can DELETE any author / series / tag / book_file

- **Severity**: HIGH
- **Impact category**: DATA_LOSS
- **Location**: `internal/handlers/authors.go:222`, `internal/handlers/series.go:219`, `internal/handlers/tags.go:160`, `internal/handlers/book_files.go:133`
- **Trigger condition**: Any authenticated user calls DELETE on any of these endpoints.
- **Consequence**: Same shape as Finding 4 — shared metadata wiped for all users. Tag/author delete cascade removes book associations; book_file delete removes download history.
- **Verdict**: CONFIRMED

#### What happens

1. Bob calls `DELETE /api/tags/{id}` for any tag.
2. No admin gate; cascade removes `book_tags` rows for EVERY user.
3. Books tagged with that tag become invisible in tag-filtered views.
4. Same flow applies to author/series/book_file.

#### Root cause

Same root cause as Finding 4 — shared resources missing admin gate. Pattern was applied to libraries but not to the rest of the books domain.

#### Evidence

- `internal/handlers/authors.go:222` — `deleteAuthor` no `requireAdmin`.
- `internal/handlers/series.go:215–225` — `deleteSeries` no `requireAdmin`.
- `internal/handlers/tags.go:156–166` — `deleteTag` no `requireAdmin`.
- `internal/handlers/book_files.go:133` — `deleteBookFile` no `requireAdmin`.
- Routes: `internal/server/routes.go:75–84,103` — all `s.requireAuth`.

#### How to verify

`curl -X DELETE -H "Cookie: biblioteka_token=B" http://localhost:8080/api/tags/<any-tag-id>` from a non-admin account; observe 204.

#### Suggested fix

Add `if !requireAdmin(h.DB, w, r) { return }` to each of the four delete handlers. Single PR with Finding 4.

#### Confidence

- Verified handler bodies and route middleware. Proof: N/A.

#### Gate evidence

- IMPACT_CATEGORY: DATA_LOSS (multi-user).
- IS_LIVE_SURFACE: standard CRUD endpoints.
- NO_SCENARIO_MITIGATION: no admin gate; books-domain delete diverges from library-domain pattern.
- CTO_TEST: yes — paired with Finding 4.

#### Labels

authz, shared-resource, data-loss, impact:data_loss

---

### Finding 6: ExifTool argument injection via newline in uploaded filename — permanent metadata-extractor death plus potential arbitrary file read

- **Severity**: HIGH
- **Impact category**: PROD_INCIDENT (extractor death) + potential SECURITY_BREACH (arbitrary `-tagsFromFile` reads)
- **Location**: `internal/handlers/book_upload.go:128` (filename sanitization) + `internal/exif/exif.go:250` (stdin write)
- **Trigger condition**: Any authenticated user uploads a file with a multipart filename containing `\n`, e.g. `filename*=UTF-8''book%0A-tagsFromFile%0A/etc/passwd%0A-execute.epub`.
- **Consequence**: Newline survives sanitization, gets written to ExifTool's stay-open stdin protocol where each newline-separated line is a distinct argument. Minimum outcome: extractor's `markDead()` fires permanently (Finding 7); upper bound: attacker injects ExifTool flags (`-tagsFromFile`, `-overwrite_original`) to read attacker-chosen files or mutate on-disk metadata.
- **Verdict**: CONFIRMED (proof PASSED for newline preservation through both `filepath.Base` and Go's multipart RFC 5987 parser)

#### What happens

1. Attacker POSTs upload with `Content-Disposition: form-data; name="file"; filename*=UTF-8''book%0A-tagsFromFile%0A/etc/passwd%0A-execute.epub`.
2. Go's multipart parser decodes the `%0A` to a real newline; `filepath.Base("book\n-tagsFromFile\n/etc/passwd\n-execute.epub")` returns the same string (Base only strips path separators).
3. Staging path concatenates `prefix + "_" + filename`, preserving the newlines.
4. `process:file` worker calls `ExtractMetadataFromFile(stagingPath)`; `internal/exif/exif.go:250` writes the path via `fmt.Fprintln(e.stdin, file)`.
5. ExifTool's `-stay_open True -@ -` protocol treats each newline-separated stdin line as a separate argument until `-execute`.
6. ExifTool now sees `book`, `-tagsFromFile`, `/etc/passwd`, `-execute` — runs with attacker-controlled flags. Best case: protocol desyncs and `markDead()` fires permanently. Worst case: reads/overwrites attacker-chosen files.

#### Root cause

Filename sanitization assumes only path separators are dangerous, but newlines control the ExifTool stay-open protocol framing.

#### Evidence

- Sanitization: `internal/handlers/book_upload.go:128` — `filename := filepath.Base(header.Filename)` (preserves `\n`).
- Stdin write: `internal/exif/exif.go:250` — `fmt.Fprintln(e.stdin, file)`.
- Scanner also accepts filenames with newlines: `internal/jobs/scan_directory.go:119–131` via `filepath.Abs` (no strip).
- Proof script: `/tmp/proof_finding4.go` confirmed `filepath.Base("foo\nbar.epub") == "foo\nbar.epub"`; `/tmp/proof_multipart2.go` confirmed Go's multipart accepts RFC 5987 `%0A`.

#### How to verify

```bash
# Construct a multipart upload with a newline filename
curl -X POST -H "Cookie: biblioteka_token=..." \
  -F "file=@book.epub;filename*=UTF-8''book%0A-tagsFromFile%0A/etc/passwd%0A-execute.epub" \
  -F "library_id=..." \
  http://localhost:8080/api/books/upload
```

Observe the worker log for ExifTool error or `markDead` warning.

#### Suggested fix

In `internal/handlers/book_upload.go:128` and `internal/jobs/scan_directory.go:119`, reject filenames containing `\n`, `\r`, or any control char (`strings.ContainsAny(filename, "\x00\r\n")` → reject with 400). Alternatively, in `internal/exif/exif.go:250`, switch to a binary-safe path-passing mechanism or URL-encode the path before writing.

#### Confidence

- Verified by proof scripts (both `filepath.Base` and Go's multipart).
- Verified ExifTool's `-stay_open` protocol semantics from documentation.
- Proof script result: PASSED.

#### Gate evidence

- IMPACT_CATEGORY: PROD_INCIDENT (permanent extractor death); upper bound SECURITY_BREACH.
- IS_LIVE_SURFACE: any authenticated user can upload.
- NO_SCENARIO_MITIGATION: no newline strip; no validation downstream.
- CTO_TEST: yes — single malicious upload disables a primary feature instance-wide.

#### Labels

filename-injection, exiftool, argv-split, impact:prod_incident

---

### Finding 7: One malformed file permanently disables metadata extraction across the entire instance

- **Severity**: HIGH
- **Impact category**: PROD_INCIDENT
- **Location**: `internal/exif/exif.go:122–126` (scanner default), `:267–291` (markDead on any error), `:218–219` (ErrDead block); `internal/metadata/extractor.go:32–39` (no buffer option, no restart)
- **Trigger condition**: Any file whose ExifTool TSV output exceeds 64 KiB for a single record (large embedded XMP, verbose metadata via `-ee3 -api requestall=3`), OR a file that crashes ExifTool, OR Finding 6's exploitation.
- **Consequence**: `markDead()` fires permanently. Every subsequent `ExtractMetadataFromFile` returns `ErrDead`. All future ingestion silently falls back to filename-only metadata for the rest of the server's life. Recovery requires process restart.
- **Verdict**: CONFIRMED

#### What happens

1. A file produces > 64 KiB ExifTool output (default `bufio.Scanner` MaxScanTokenSize).
2. `bufio.ErrTooLong` propagates; `markDead()` runs; closes stdin; waits for cmd.
3. The single shared `*metadata.Extractor` (created once at `cmd/server/main.go:98`, registered with the only `ProcessFileHandler`) is now permanently dead.
4. All 4 worker goroutines (default `Concurrency=4`) get `ErrDead` on every subsequent metadata call.
5. Users see books appear without titles/authors/covers; SSE clients see no progress events.

#### Root cause

`NewExiftool(ctx)` called with NO options at `internal/metadata/extractor.go:32–39` — `bufferSet=false` so the scanner uses the default 64 KiB max. `markDead` is called on any scanner error and sets the dead flag permanently with no restart path.

#### Evidence

- No buffer override: `internal/metadata/extractor.go:32–39`.
- Default scanner: `internal/exif/exif.go:122–126`.
- markDead on any error: `internal/exif/exif.go:267–291`.
- Permanent block: `internal/exif/exif.go:218–219`.
- `grep -r "Restart" internal/exif internal/metadata` → empty.

#### How to verify

Construct an EPUB whose ExifTool output exceeds 64 KiB (large embedded XMP block, dozens of contributors, etc.) and upload it. Subsequent uploads silently get no metadata.

#### Suggested fix

Two parts:
1. Pass `WithBuffer(maxScanTokenSize)` to `NewExiftool` with a much larger ceiling (e.g., 4 MiB).
2. Add a `Restart()` method on `metadata.Extractor` that the worker invokes when `ExtractMetadataFromFile` returns `ErrDead`, respawning the subprocess.

#### Confidence

- Verified default scanner buffer (Go stdlib `bufio.MaxScanTokenSize=64KiB`).
- Verified no restart logic via grep.
- Proof script result: N/A (verified by code).

#### Gate evidence

- IMPACT_CATEGORY: PROD_INCIDENT — instance-wide silent denial of a primary feature.
- IS_LIVE_SURFACE: single shared Extractor across all workers.
- NO_SCENARIO_MITIGATION: no respawn, no supervisor.
- CTO_TEST: yes — silent metadata-extraction outage is a sprint-stopper.

#### Labels

pipeline-death, subprocess, no-restart, impact:prod_incident

---

### Finding 8: Crafted MOBI/AZW3 cover causes worker OOM kill

- **Severity**: HIGH
- **Impact category**: PROD_INCIDENT
- **Location**: `internal/exif/mobi_cover.go:48`
- **Trigger condition**: Any authenticated user uploads a MOBI/AZW3 whose embedded cover record points at a tiny PNG/GIF with declared dimensions like `65535×65535`.
- **Consequence**: `image.Decode` allocates a pixel buffer sized by header dimensions × 4 bytes (RGBA) — ~16 GiB for the header above. Worker OOMs. Process restart loop. With `Concurrency=4` and shared Extractor, this can wedge processing.
- **Verdict**: CONFIRMED

#### What happens

1. Attacker constructs a MOBI containing a tiny encoded PNG/JPEG with header dimensions of 65535×65535 (well under 1 KB compressed).
2. Attacker uploads.
3. Worker's `GetMobiCover` calls `image.Decode(io.LimitReader(f, coverlength))`. The `io.LimitReader` only caps encoded bytes, not the decoded pixel buffer.
4. Decoder allocates ~16 GiB. Linux OOM-killer terminates the worker. Process restarts and tries again on the same job (asynq retry).

#### Root cause

Cover-size guard is byte-based (`coverutil.DecodeDataURL` enforces 20 MB on encoded bytes), but applies only to data-URL covers extracted via ExifTool. MOBI's raw-byte path goes directly to `image.Decode` with no `image.DecodeConfig` dimension check.

#### Evidence

- Buggy path: `internal/exif/mobi_cover.go:48` — `image.Decode(io.LimitReader(f, coverlength))`.
- `coverutil.DecodeDataURL` only protects data-URL paths (`internal/coverutil/decode.go`).

#### How to verify

Build a malicious MOBI (e.g., via Python `mobi`/`KindleUnpack`) with a PNG IHDR width=65535 height=65535 in the cover record. Upload it. Observe worker memory spike + OOM.

#### Suggested fix

In `internal/exif/mobi_cover.go`, call `image.DecodeConfig` first; reject covers where `width*height > maxPixelBudget` (e.g., 50 megapixels):

```go
cfg, _, err := image.DecodeConfig(io.LimitReader(f, coverlength))
if err != nil { return nil, err }
if cfg.Width*cfg.Height > maxPixelBudget {
    return nil, fmt.Errorf("cover dimensions too large: %dx%d", cfg.Width, cfg.Height)
}
// ... then seek back and decode
```

#### Confidence

- Verified `image.Decode` has no built-in dimension check.
- Verified `coverutil` only protects data-URL path.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: PROD_INCIDENT — worker OOM crash from a single upload.
- IS_LIVE_SURFACE: any authenticated user upload of MOBI/AZW3.
- NO_SCENARIO_MITIGATION: byte cap is not a dimension cap.
- CTO_TEST: yes — worker crash from user input is sprint-stopping.

#### Labels

decoder-bomb, oom, image-decode, impact:prod_incident

---

### Finding 9: Reading streak and active-days metrics undercount the most common reading pattern — punish users for reading one book at a time

- **Severity**: HIGH
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/db/reading_progress.go:217–228` (active days), `:66–81` (streak); schema `db/migrations/sqlite/20260317000001_create_kosync_tables.sql:25`; upsert `internal/db/kosync.go:88–94`
- **Trigger condition**: User reads any single book across multiple days via KOReader (KOSync sync). Default for sequential readers.
- **Consequence**: Dashboard shows `1-day streak` and "1 days reading" for a user who read every day for 30 days. The metric punishes the most-common reading behavior it should celebrate.
- **Verdict**: CONFIRMED

#### What happens

1. User reads Book A on Mon, Tue, Wed (3 KOSync syncs).
2. `reading_progress` has UNIQUE(user_id, document); upsert overwrites `updated_at` on every sync.
3. After Wednesday, there's ONE row with `updated_at = Wednesday`.
4. `ActiveDays = COUNT(DISTINCT DATE(updated_at))` for the week = 1.
5. Dashboard "Reading Activity" card shows "1 days reading"; streak shows 1.

#### Root cause

`reading_progress` is current-state, not history. The schema, upsert, and aggregate queries together guarantee the metric is wrong for any user reading sequentially.

#### Evidence

- Schema: `db/migrations/sqlite/20260317000001_create_kosync_tables.sql:25` — `UNIQUE (user_id, document)`.
- Upsert: `internal/db/kosync.go:88–94` — `ON CONFLICT … DO UPDATE SET … updated_at = NOW()`.
- Query: `internal/db/reading_progress.go:217–228` — counts distinct `DATE(updated_at)`.
- Test confirms: `internal/db/reading_progress_test.go:445–489` works only when test uses different document IDs per day.
- UI: `frontend/src/components/Dashboard.svelte:289,439–460`.

#### How to verify

```sql
-- Inspect the schema
.schema reading_progress
-- Insert two updates for the same document on different days
INSERT INTO reading_progress (...) VALUES (...);
UPDATE reading_progress SET updated_at = '2026-06-02' WHERE document = X;
UPDATE reading_progress SET updated_at = '2026-06-03' WHERE document = X;
-- Run ActiveDays query → returns 1
```

#### Suggested fix

Add an append-only `reading_events(id, user_id, document, occurred_at, percentage)` table written from `UpsertReadingProgress`. Compute `ActiveDays`, `CurrentStreak`, `LongestStreak` from distinct dates in that table. Migration backfills are unnecessary — start fresh; users see correct numbers going forward.

#### Confidence

- Verified schema, upsert, query, and UI rendering.
- Proof script result: N/A (verified by code + tests).

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — primary dashboard metric wrong for common pattern.
- IS_LIVE_SURFACE: Dashboard front page.
- NO_SCENARIO_MITIGATION: no history table, no aggregation by created_at.
- CTO_TEST: yes — flagship "Year in Books" feature is fundamentally wrong.

#### Labels

display-truth, stats, schema-gap, impact:support_burden

---

### Finding 10: Year-in-Books and reading stats show zero for Kobo-only users with extensive reading activity

- **Severity**: HIGH
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/db/reading_progress.go:203–228`; `internal/db/recommendations.go:25–30`; `frontend/src/components/Dashboard.svelte:271–278`
- **Trigger condition**: User syncs reading state via the Kobo sync protocol only (no KOReader/KOSync).
- **Consequence**: Year-in-Books card shows `0 books finished`, `0 longest streak`, `0 days reading` even after the user has finished hundreds of books. Reading Activity card says "No reading activity recorded yet. Connect KOReader via Settings → KOSync..." — wrong for the entire Kobo-only user class.
- **Verdict**: CONFIRMED

#### What happens

1. Kobo user has hundreds of `kobo_reading_states` rows with `status='Finished'`.
2. Dashboard loads `GetYearInBooks` — queries only `reading_progress` (KOSync).
3. Returns zeros.
4. The card renders anyway because `Dashboard.svelte:397` triggers on `books_finished > 0 || active_days > 0 || total_downloads > 0` — `total_downloads` is non-zero, so the user sees a card full of zeros instead of getting hidden.

#### Root cause

Two parallel reading-state systems: `reading_progress` (KOSync) and `kobo_reading_states` (Kobo). Stats query only the first; recommendations query only the second. No unification.

#### Evidence

- Stats only `reading_progress`: `internal/db/reading_progress.go:203–228`.
- Recommendations only `kobo_reading_states`: `internal/db/recommendations.go:25–30`.
- Hard-coded KOSync-only copy: `frontend/src/components/Dashboard.svelte:271–278`.

#### How to verify

Use the Kobo sync endpoints to mark books as finished (`kobo_reading_states.status='Finished'`); open the dashboard; observe Year-in-Books shows zeros.

#### Suggested fix

Either:
- (a) Write to both tables on every reading-state update; or
- (b) UNION ALL both sources in aggregate queries (`reading_progress` + `kobo_reading_states`); or
- (c) Migrate to a unified `reading_events` table (also addresses Finding 9).

(c) is the cleanest long-term direction; (b) is the smallest immediate fix.

#### Confidence

- Verified by reading queries and dashboard render conditions.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — primary metric wrong for entire user class.
- IS_LIVE_SURFACE: Dashboard.
- NO_SCENARIO_MITIGATION: no UNION; no dual-write.
- CTO_TEST: yes — Kobo users seeing "0 books finished" after finishing 30 is a flagship-feature failure.

#### Labels

display-truth, dual-system, kobo, impact:support_burden

---

### Finding 11: "Currently Reading" widget displays opaque KOReader hashes instead of book titles

- **Severity**: HIGH
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/handlers/reading_progress.go:22,98`; schema `db/migrations/sqlite/20260317000001_create_kosync_tables.sql:13–26`; UI `frontend/src/components/Dashboard.svelte:317–353`
- **Trigger condition**: Any KOReader user opens the Dashboard.
- **Consequence**: "Currently Reading" list shows entries like `0b3e72cb1c1c2bca9d8e... 35%` — raw KOReader MD5 document hashes as the visible title, with no author, cover, or link. The widget is unusable as-shipped.
- **Verdict**: CONFIRMED

#### What happens

1. KOReader user has reading progress for several books.
2. `HandleReadingProgressStats` returns `Document: p.Document` raw (the opaque KOReader hash).
3. `Dashboard.svelte:317–353` renders `{item.document}` directly as the primary label.

#### Root cause

`reading_progress` stores an opaque KOReader document hash with no `book_id` column or hash→book mapping. There is no resolution from `document` to `books.title`.

#### Evidence

- Raw DTO: `internal/handlers/reading_progress.go:22,98` — `Document` carries unchanged hash.
- Schema: `db/migrations/sqlite/20260317000001_create_kosync_tables.sql:13–26` — no `book_id` column.
- UI: `frontend/src/components/Dashboard.svelte:317–353` — renders `{item.document}`.

#### How to verify

Sync a KOReader; open the Dashboard; observe the Currently Reading list.

#### Suggested fix

Either:
- (a) Add `book_id` to `reading_progress` resolved by hashing book files on import/scan (store MD5 in `book_files`, join on hash); or
- (b) Compute and persist a hash→book_id map in a separate table at scan time; or
- (c) At minimum, render a friendlier label (first 8 chars + ellipsis) until (a) ships.

#### Confidence

- Verified DTO, handler, schema, and UI rendering.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — primary widget shows gibberish.
- IS_LIVE_SURFACE: Dashboard.
- NO_SCENARIO_MITIGATION: no title resolution anywhere in the path.
- CTO_TEST: yes — widget unusable as-shipped.

#### Labels

display-truth, schema-gap, impact:support_burden

---

### Finding 12: First-user-admin race lets a second concurrent signup also become admin

- **Severity**: MEDIUM
- **Impact category**: SECURITY_BREACH
- **Location**: `internal/db/users.go:53–73` (`CreateUser`); also `:79–99` (`CreateOIDCUser`)
- **Trigger condition**: Two near-simultaneous signups on a fresh install with no `INITIAL_ADMIN_*` env vars set.
- **Consequence**: Both signups read `COUNT(*) FROM users = 0`, both insert with `is_admin=true`. Attacker who races the legitimate operator's first signup gets full admin authority over a server they don't own.
- **Verdict**: CONFIRMED (validator downgraded CRITICAL→MEDIUM citing narrow race window)

#### What happens

1. Operator brings up a fresh Biblioteka instance (Docker pull → first run).
2. Attacker hits `/api/auth/signup` simultaneously with the operator.
3. Both requests: `SELECT EXISTS(email=...)` → false; `SELECT COUNT(*) FROM users` → 0; INSERT with `is_admin=true`.
4. Both users are admin. Attacker has full config access.

#### Root cause

`CreateUser` issues three separate `QueryRowContext` calls (EXISTS, COUNT, INSERT) without `BeginTx`, advisory lock, or `INSERT…WHERE NOT EXISTS`. No schema partial unique index on `is_admin`.

#### Evidence

- `internal/db/users.go:53–73` — three separate queries, no transaction.
- `internal/db/users.go:79–99` — `CreateOIDCUser` structurally identical, same race via OIDC callback.
- Schema: `db/migrations/sqlite/20260224000000_add_is_admin_to_users.sql:1–2` — no partial unique index.

#### How to verify

```bash
# Spin up a fresh instance; immediately fire two concurrent signups
( curl -X POST -d '{"email":"a@x","password":"...","name":"a"}' http://localhost:8080/api/auth/signup ) &
( curl -X POST -d '{"email":"b@x","password":"...","name":"b"}' http://localhost:8080/api/auth/signup ) &
wait
sqlite3 data/biblioteka.db "SELECT email, is_admin FROM users;"
```

Two `is_admin=1` rows = bug confirmed.

#### Suggested fix

Use a single SQL statement to make first-user-admin atomic:

```sql
INSERT INTO users (..., is_admin)
VALUES (..., NOT EXISTS (SELECT 1 FROM users))
RETURNING ...
```

Or wrap the count+insert in a `BeginTx(SERIALIZABLE)`. Add a defensive partial unique index `CREATE UNIQUE INDEX users_single_auto_admin ON users ((TRUE)) WHERE is_admin AND <bootstrap_marker>` for defense in depth.

#### Confidence

- Verified by reading the function body; no transaction, no constraint.
- Proof script result: N/A (requires concurrent test setup).

#### Gate evidence

- IMPACT_CATEGORY: SECURITY_BREACH — admin privilege escalation.
- IS_LIVE_SURFACE: `/api/auth/signup` public endpoint.
- NO_SCENARIO_MITIGATION: no transaction, no constraint.
- CTO_TEST: yes — silent admin grant is a sprint-blocker for any self-hostable product.

#### Labels

auth-race, first-user-admin, signup, impact:security_breach

---

### Finding 13: SMTP send bypasses SSRF protection via DNS rebinding

- **Severity**: MEDIUM
- **Impact category**: SECURITY_BREACH
- **Location**: `internal/smtp/config.go:140–152` (validation), `internal/smtp/send.go:65` (raw Dialer)
- **Trigger condition**: Admin sets `SMTP_HOST=attacker.example.com` (public IP at validation time). Attacker's DNS later rebinds to 127.0.0.1 or 169.254.169.254.
- **Consequence**: SMTP send connects to the rebound internal IP. SMTP banner-exchange leaks internal-network reachability information; STARTTLS errors in test-email responses leak internal-service banners.
- **Verdict**: CONFIRMED-DOWNGRADED to MEDIUM (validator: SMTP protocol leakage is more limited than full HTTP SSRF; requires admin trust to configure host)

#### What happens

1. Admin configures `SMTP_HOST=smtp.attacker.example.com`.
2. `ValidateHost` only rejects literal private IPs and "localhost"; no DNS resolution.
3. On send, `smtp.Send` uses raw `net.Dialer{Timeout: 10s}`; DNS resolves at dial time.
4. Attacker flips DNS to `169.254.169.254`. Server connects there and speaks SMTP. Connection success/failure pattern + STARTTLS banner content leaks internal info to the admin via test-email error responses.

#### Root cause

Recent fix `a3bd1640` added literal-IP/localhost rejection at config time but didn't address DNS rebinding at dial time. OIDC and Ollama paths use `ssrf.SafeHTTPClient` (connection-time IP re-validation); SMTP should mirror this with a `ssrf.SafeDialer`.

#### Evidence

- Literal-only check: `internal/smtp/config.go:140–152`.
- Vulnerable dial: `internal/smtp/send.go:65` (raw `net.Dialer`, used at lines 70, 82, 97).
- Pattern to mirror: `internal/ssrf/ssrf.go:67–101` (`SafeHTTPClient` DialContext re-resolves).

#### How to verify

Run an attacker DNS for an SMTP_HOST domain that returns a public IP at config save and `169.254.169.254` later. Send a test email; observe the SMTP attempt to the metadata-service IP.

#### Suggested fix

Implement `ssrf.SafeDialer` that re-resolves and validates IP inside the SMTP dial step. Apply it in `internal/smtp/send.go:65,70,82,97` in place of the bare `net.Dialer`.

#### Confidence

- Verified validation is literal-IP only.
- Verified dial uses raw net.Dialer.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SECURITY_BREACH — SSRF surface.
- IS_LIVE_SURFACE: admin-controlled but reachable on every send.
- NO_SCENARIO_MITIGATION: validation doesn't cover runtime rebinding.
- CTO_TEST: yes — SSRF in a service that interacts with attacker-controllable DNS is sprint-worthy.

#### Labels

ssrf, dns-rebinding, smtp, impact:security_breach

---

### Finding 14: Raw Kobo tokens leak into DEBUG access logs

- **Severity**: MEDIUM
- **Impact category**: SECURITY_BREACH
- **Location**: `internal/handlers/middleware/logging.go:73,87`
- **Trigger condition**: Operator enables DEBUG logging while Kobo sync runs.
- **Consequence**: Every Kobo device request emits two DEBUG slog entries containing the raw token in the URL. Logs commonly ship to long-retention aggregators (Loki, Datadog, ELK) where many teammates can see them.
- **Verdict**: CONFIRMED

#### What happens

Same root cause as Finding 3 — global middleware records the raw URL before the Kobo middleware strips the token. Gated by DEBUG level, which operators routinely enable for troubleshooting.

#### Root cause

`LoggingMiddleware` is in the global chain (`server.go:223`), before the mux that contains the Kobo middleware path rewrite. Both `Incoming request` and `Request completed` log entries include `slog.String(otelkeys.URL, r.URL.String())` — the raw URL containing the token.

#### Evidence

- `internal/handlers/middleware/logging.go:73,87` — `slog.String(otelkeys.URL, r.URL.String())`.
- `internal/server/server.go:223` — chain order.

#### How to verify

Set `LOG_LEVEL=debug`; hit `/kobo/<token>/v1/library/sync`; observe token in stdout.

#### Suggested fix

Same as Finding 3 — replace `r.URL.String()` with a route-templated path for `/kobo/` paths. Single shared helper.

#### Confidence

- Verified by reading the middleware and chain order.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SECURITY_BREACH — token-in-logs.
- IS_LIVE_SURFACE: any operator with DEBUG on.
- NO_SCENARIO_MITIGATION: no scrubbing.
- CTO_TEST: yes — credential-in-logs is sprint-worthy.

#### Labels

token-leak, logging, kobo, impact:security_breach

---

### Finding 15: Disabling registration locks out existing OIDC-only users

- **Severity**: MEDIUM
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/handlers/auth_compat.go:80–92`
- **Trigger condition**: Admin sets `registration_disabled=true` (intent: "no new users"). Existing OIDC user signs in.
- **Consequence**: The wrapper returns 403 "signup is disabled" BEFORE delegating to goauth (which would have found the existing user). Existing OIDC-only users can't fall back to password login (their `password_hash` is empty). If admin uses OIDC exclusively, recovery requires direct DB edits.
- **Verdict**: CONFIRMED

#### What happens

1. Admin sets `registration_disabled=true`.
2. Admin's JWT expires; admin signs in via OIDC.
3. `OIDCHandler.Callback` wrapper at `auth_compat.go:80–92` sees the gate, returns 403, short-circuits before goauth's `FindByOIDCSubject`.
4. Admin is locked out. Same for any existing OIDC user.

#### Root cause

Wrapper gates ALL OIDC callbacks instead of only the user-creation branch. The id_token's `sub` claim is available at callback time and could be used to distinguish new from returning users.

#### Evidence

- Gate: `internal/handlers/auth_compat.go:80–92`.
- Login rejects empty hash: `goauth@v0.6.1/handler/auth.go:170–174`.
- User lookup short-circuited: `goauth@v0.6.1/handler/oauth2_common.go:107–118` (`findOrCreateUser`).
- Test only covers blocked-callback case: `auth_compat_test.go:211–223`.

#### How to verify

1. Configure OIDC; create a user via OIDC login.
2. Set `registration_disabled=true`.
3. Sign out; sign in again via OIDC → 403.

#### Suggested fix

In the wrapper at `auth_compat.go:80–92`, decode the id_token (or call goauth with a UserStore decorator) and gate only the `CreateOIDCUser` path, not the entire callback:

```go
// Pseudocode
sub := idToken.Subject
if _, err := h.DB.GetUserByOIDCSubject(ctx, sub); err == nil {
    return h.OIDCHandler.Callback(w, r) // existing user, pass through
}
if registrationDisabled { return 403 }
return h.OIDCHandler.Callback(w, r) // new user, allowed
```

#### Confidence

- Verified wrapper, goauth login path, and test coverage gap.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — admin lockout requires DB surgery to recover.
- IS_LIVE_SURFACE: every OIDC-enabled deployment that ever toggles the gate.
- NO_SCENARIO_MITIGATION: no per-user-type discrimination.
- CTO_TEST: yes — admin lockout on a documented admin action.

#### Labels

oidc, registration-gate, lockout, impact:support_burden

---

### Finding 16: Two admins concurrently demoting each other can leave zero admins (instance unrecoverable)

- **Severity**: MEDIUM
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/handlers/admin.go:163–166` (self-demotion guard); `internal/db/users.go:151–167` (`SetAdmin` plain UPDATE)
- **Trigger condition**: Two admins concurrently call `PUT /api/admin/users/{otherAdminId}` with `is_admin=false`, OR one admin demotes the second who then demotes the first.
- **Consequence**: Zero admins remain; instance config is locked. Recovery requires direct DB access.
- **Verdict**: CONFIRMED

#### What happens

1. Admins A and B both currently exist.
2. A demotes B; B (still admin) demotes A. Both succeed.
3. Or both run concurrent demotion PUTs. Both pass the self-check; both UPDATEs commit.
4. No admin remains.

#### Root cause

`HandleSetAdmin` only blocks self-demotion. `SetAdmin` is `UPDATE users SET is_admin=$1 WHERE id=$2` with no count-of-remaining-admins check.

#### Evidence

- Self-only guard: `internal/handlers/admin.go:163–166`.
- Plain UPDATE: `internal/db/users.go:151–167`.
- No `CountAdmins` analogue.

#### How to verify

```bash
# As admin A
curl -X PUT -d '{"is_admin":false}' http://localhost:8080/api/admin/users/B_id
# As admin B (still admin if A's request hasn't propagated)
curl -X PUT -d '{"is_admin":false}' http://localhost:8080/api/admin/users/A_id
```

#### Suggested fix

In a transaction, before demoting:

```sql
BEGIN;
SELECT COUNT(*) FROM users WHERE is_admin AND id != $1; -- must be >= 1
UPDATE users SET is_admin=$2 WHERE id=$1;
COMMIT;
```

Reject the demotion with 409 if it would leave zero admins.

#### Confidence

- Verified handler and DB function.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — total admin lockout requires DB surgery.
- IS_LIVE_SURFACE: admin endpoint.
- NO_SCENARIO_MITIGATION: no count-of-admins check.
- CTO_TEST: yes — one-SQL-guard fix; recovery cost is high.

#### Labels

admin-zero, race, impact:support_burden

---

### Finding 17: Orphan file in library tree after DB write failure during ingest

- **Severity**: MEDIUM
- **Impact category**: DATA_LOSS
- **Location**: `internal/jobs/process_book_file.go:109–120` (order of operations); `internal/jobs/book_record_helpers.go:117–200` (no compensating move-back); `internal/jobs/book_path_helpers.go:118–145` (recovery candidates wrong)
- **Trigger condition**: Transient DB error (lock contention, timeout, FK violation) during `CreateBookWithFile` AFTER the file has been moved to its reorganized location.
- **Consequence**: File sits at `<libRoot>/Author/Title/file.epub` with NO `books` row, NO `book_files`, NO sidecar, NO library link. Asynq retries can't find it because they look at the original staging path. Only recovered if a future 24h `scan:libraries` walks the file (and the library is monitored).
- **Verdict**: CONFIRMED

#### What happens

1. `process:file` moves a staged upload to its reorganized destination.
2. `CreateBookWithFile` fails transiently.
3. asynq retries. `resolveSourcePath` does `os.Stat(p.Path)` on the original staging path → file gone.
4. Falls back to candidates derived from `pathparser.ParseBookPath(p.Path, libraryRoot)` — but `p.Path` is `<libRoot>/.uploads/<hex>_file.epub`, so the parsed Author="uploads" Title="<hex>_file" doesn't match the real reorganized destination.
5. Returns "source file no longer exists, skipping"; every retry skips silently.

#### Root cause

Reorganize is not atomic with the DB write, and the recovery logic doesn't know the post-move path.

#### Evidence

- Order: `internal/jobs/process_book_file.go:109–120`.
- No compensation: `internal/jobs/book_record_helpers.go:117–200`.
- Wrong recovery candidates: `internal/jobs/book_path_helpers.go:118–145`.

#### How to verify

Inject a transient DB error during `CreateBookWithFile` (e.g., temporarily revoke write perms on the books table). Upload a file. Observe the file in the library tree with no DB row after retries exhaust.

#### Suggested fix

On `CreateBookWithFile` failure, move the file back to its staging path (compensating action) before returning the error to asynq. Or persist the destination path in the asynq payload after move so retries know where to look.

#### Confidence

- Verified order of operations and recovery logic.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: DATA_LOSS — silent orphan accumulation.
- IS_LIVE_SURFACE: every upload + scan.
- NO_SCENARIO_MITIGATION: pathparser fallback can't reconstruct the destination.
- CTO_TEST: yes — silent data-integrity hole.

#### Labels

orphan-file, compensation, saga, impact:data_loss

---

### Finding 18: scan:libraries walks `.uploads/` directory, racing with in-flight uploads to produce duplicate book rows

- **Severity**: MEDIUM
- **Impact category**: DATA_LOSS
- **Location**: `internal/jobs/scan_directory.go:82–147`
- **Trigger condition**: A staged upload sits in `<libRoot>/.uploads/` while a 24h `scan:libraries` cron fires (or watch-folder cron every 1m).
- **Consequence**: Scanner enqueues a SECOND `process:file` task for the staged path with a DIFFERENT payload shape (`UserID=""`, no overrides), so asynq dedup misses. Two workers race the same staged file; both can pass `checkDuplicate`; one wins the rename, the loser falls back to staging path with a stale file reference. Two `books` rows + a broken `book_file` pointer.
- **Verdict**: CONFIRMED

#### What happens

1. User uploads a 500 MB file at 14:00. `process:file` enqueued; staged file sits in `.uploads/`.
2. 14:30: cron `scan:watch-folder` (every 1m) or daily `scan:libraries` fires.
3. `ScanDirectory` walks `<libRoot>` unconditionally — no skip for `.uploads/`. Enqueues `process:file` with default payload.
4. Asynq dedup keyed on serialized payload; different fields → both jobs enter the queue.
5. Workers race; both pass `checkDuplicate` because the row doesn't exist yet; both extract metadata. One wins the rename, loser falls back to staging path and inserts a `book_files` row pointing at the just-moved location.
6. End state: two `books` rows + one broken `book_file.file_path`.

#### Root cause

`ScanDirectory` walks unconditionally. Upload and scan payloads diverge, so asynq dedup doesn't catch the duplicate.

#### Evidence

- `internal/jobs/scan_directory.go:82–147` — `filepath.WalkDir`, no `.uploads` skip.
- Different payload shapes: upload includes UserID + overrides; scan does not.
- Race in fallback: `internal/jobs/book_record_helpers.go:21–105` — rename-fail fallback uses original path without fresh duplicate check.

#### How to verify

Upload a large file (slow processing); during processing, run `make scan` or trigger `scan:libraries` manually. Observe two book rows for the same file.

#### Suggested fix

In `ScanDirectory`, skip the staging dir:

```go
filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
    if d != nil && d.IsDir() && d.Name() == ".uploads" {
        return fs.SkipDir
    }
    // ...
})
```

Optional: normalize the `process:file` payload so asynq dedup catches the race.

#### Confidence

- Verified walker behavior and payload shapes.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: DATA_LOSS (duplicate rows + broken pointer).
- IS_LIVE_SURFACE: every upload while watch-folder cron is on.
- NO_SCENARIO_MITIGATION: dedup key differs; no skip; rename fallback doesn't re-check.
- CTO_TEST: yes — silent corruption under normal usage.

#### Labels

scan-race, duplicate-rows, process-file, impact:data_loss

---

### Finding 19: Deleting a book leaves the file orphaned on disk — admin "cleanup" doesn't reclaim space

- **Severity**: MEDIUM
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/db/books.go:252–255`; `internal/handlers/book_files.go:133–141`
- **Trigger condition**: Admin deletes a book or book_file through the UI.
- **Consequence**: DB rows are removed; file on disk remains forever. `du -sh /library` shows zero change after deleting 50 books. No garbage collection job exists.
- **Verdict**: CONFIRMED

#### What happens

1. Admin DELETE /api/books/{id} → handler calls `deleteResource` → calls `DeleteBook`.
2. `DeleteBook` is `DELETE FROM books WHERE id=$1` only. ON DELETE CASCADE removes related DB rows.
3. The actual EPUB/PDF/MOBI file plus sidecar OPF and cover image remain on disk.
4. On next 24h scan, those files get re-indexed as new books.

#### Root cause

Delete handlers only delete DB rows. No `os.Remove` in any delete path. No janitor job.

#### Evidence

- `internal/db/books.go:252–255` — DB-only delete.
- `internal/handlers/book_files.go:133–141` — `deleteResource` with no disk cleanup.
- `grep -rn "os.Remove" internal/handlers/book*.go` returns only upload-staging cleanup paths.
- `cmd/server/main.go:138–146` — only `scan:libraries` and `scan:watch-folder` jobs; no cleanup cron.

#### How to verify

1. `du -sh data/library` → baseline.
2. Delete a book via UI.
3. `du -sh data/library` → unchanged.

#### Suggested fix

In `deleteBook` and `deleteBookFile`, capture file paths first, validate they're under a library root (`isBookFilePathAllowed`), then call `os.Remove` AFTER the DB transaction commits (best-effort; log failures). Also add a periodic GC job for `.uploads/` and orphaned files (sweeps files older than N hours not tracked by an in-flight job).

#### Confidence

- Verified by reading delete handlers and grepping the codebase.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — admin promised cleanup that doesn't happen.
- IS_LIVE_SURFACE: primary admin flow.
- NO_SCENARIO_MITIGATION: no file removal, no janitor.
- CTO_TEST: yes — admin filing a "delete doesn't free space" support ticket.

#### Labels

data-cleanup, broken-promise, impact:support_burden

---

### Finding 20: "Books Finished" stat silently migrates between years when a user re-opens a finished book

- **Severity**: MEDIUM
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/db/reading_progress.go:203–207`
- **Trigger condition**: User finished books in 2025; opens one in 2026 (or KOReader's session-resume pings).
- **Consequence**: 2025's "Books Finished" silently decrements; 2026's increments. Historical record is non-durable; "2024 in Books" looks different from one year to the next.
- **Verdict**: CONFIRMED

#### What happens

1. User finishes Book A in Dec 2025. Year-in-Books 2025 = 5 books.
2. Jan 2026: user re-opens Book A; KOReader pings; `UpsertReadingProgress` bumps `updated_at` to now.
3. Year-in-Books 2025 query (`updated_at BETWEEN 2025-01-01 AND 2025-12-31`) excludes the row.
4. Year-in-Books 2025 = 4 books. 2026 = 1 book.

#### Root cause

`BooksFinished` buckets by `updated_at`. Because `reading_progress` is current-state-overwriting, every re-open moves the row's bucketing date.

#### Evidence

- Query: `internal/db/reading_progress.go:203–207`.
- Upsert overwrites: `internal/db/kosync.go:88–94`.
- No `first_finished_at` column in any migration (grep confirms).

#### How to verify

Set `updated_at` on an old finished `reading_progress` row to a fresh timestamp; observe Year-in-Books for the original year drops by one.

#### Suggested fix

Add a `first_finished_at TIMESTAMP` column. Set it the first time `percentage >= 0.99` and never update it. Bucket `BooksFinished` by `first_finished_at` instead of `updated_at`. Backfill from `updated_at` for existing rows.

#### Confidence

- Verified query and upsert behavior.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — historical stat silently rewrites itself.
- IS_LIVE_SURFACE: Year-in-Books card on Dashboard.
- NO_SCENARIO_MITIGATION: no immutable "finished" timestamp.
- CTO_TEST: yes — users notice "but I read 5 last year" → ticket.

#### Labels

display-truth, history-rewrite, impact:support_burden

---

### Finding 21: Recommendations include books KOReader users have already finished

- **Severity**: MEDIUM
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/db/recommendations.go:25–30,92`
- **Trigger condition**: KOReader/KOSync user (no Kobo device) opens the dashboard.
- **Consequence**: The recommendation engine excludes books only from `kobo_reading_states`. KOSync-tracked reads don't filter; the engine recommends books the user finished last week.
- **Verdict**: CONFIRMED

#### What happens

1. KOReader user has 200 books in `reading_progress` with high percentages.
2. Recommendation engine's `user_reads` CTE: `SELECT book_id FROM kobo_reading_states WHERE user_id=$1 AND status IN ('Reading','Finished')` — returns empty for KOReader-only user.
3. Exclusion `WHERE NOT EXISTS (...)` is a no-op.
4. Already-finished books appear in recommendations.

#### Root cause

Same dual-system gap as Findings 10/22 — two reading-state sources never unified.

#### Evidence

- `internal/db/recommendations.go:25–30,92`.

#### How to verify

Sync several KOReader books to high progress (`percentage >= 0.99`). Open dashboard recommendations. See those same books listed.

#### Suggested fix

In `user_reads` CTE, UNION both sources:

```sql
user_reads AS (
    SELECT book_id FROM kobo_reading_states
    WHERE user_id=$1 AND status IN ('Reading','Finished')
    UNION
    SELECT bf.book_id FROM reading_progress rp
    JOIN book_files bf ON bf.file_hash = rp.document  -- requires book_id mapping (Finding 11)
    WHERE rp.user_id=$1 AND rp.percentage >= 0.05
)
```

Depends on resolving Finding 11's hash→book mapping. Cheaper short-term: persist a `document → book_id` map at upload/scan time.

#### Confidence

- Verified by reading the query.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — engine recommends already-read titles.
- IS_LIVE_SURFACE: Dashboard recommendations.
- NO_SCENARIO_MITIGATION: no UNION.
- CTO_TEST: yes — wrong-recommendation feedback erodes feature trust.

#### Labels

display-truth, recommendations, dual-system, impact:support_burden

---

### Finding 22: Group member progress hides KOReader readers from co-readers in mixed-device groups

- **Severity**: MEDIUM
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/db/reading_groups.go:312–323`
- **Trigger condition**: Reading group with at least one Kobo user and one KOReader user.
- **Consequence**: Group progress page for a shared book shows Kobo users' progress correctly. KOReader users appear as "0%, never updated" even when they're 70% through.
- **Verdict**: CONFIRMED

#### What happens

1. Reading group has 4 members reading Book B.
2. 2 use Kobo (rows in `kobo_reading_states`), 2 use KOReader (rows in `reading_progress`).
3. `ListGroupMemberProgress` LEFT JOINs `kobo_reading_states` only.
4. The 2 Kobo readers show real percentages. The 2 KOReader readers show 0%.

#### Root cause

Same dual-system gap as Findings 10/21.

#### Evidence

- `internal/db/reading_groups.go:312–323`.

#### How to verify

Create a group with two users; one syncs via Kobo (set kobo_reading_state), one syncs via KOSync (insert reading_progress for the same document). Open `/api/groups/{id}/progress?book_id=...`. Only the Kobo user appears with progress.

#### Suggested fix

Same as Finding 21 — UNION both reading-state sources, taking the max percentage and most-recent timestamp. Depends on Finding 11's hash→book mapping.

#### Confidence

- Verified by reading the query.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — co-readers can't see each other's actual progress.
- IS_LIVE_SURFACE: groups feature.
- NO_SCENARIO_MITIGATION: no UNION.
- CTO_TEST: yes — primary collaboration feature broken for the common heterogeneous-device case.

#### Labels

display-truth, groups, dual-system, impact:support_burden

---

### Finding 23: Libraries page shows empty-state ("Add A Library") when API call fails — admin creates a duplicate

- **Severity**: MEDIUM
- **Impact category**: DATA_LOSS
- **Location**: `frontend/src/stores/libraries.svelte.ts:39–51`; UI consumers `frontend/src/components/Libraries.svelte:84–91`, `Sidebar.svelte:36–39,195`, `Dashboard.svelte:97–99`
- **Trigger condition**: `GET /api/libraries` fails transiently (5xx, network blip, expired JWT).
- **Consequence**: User sees the empty-state "Add A Library" call-to-action and a blank sidebar. They conclude their libraries were deleted and create a duplicate pointing at the same folders; scanner indexes everything twice.
- **Verdict**: CONFIRMED

#### What happens

1. User has 3 libraries with thousands of books.
2. Transient API failure on `/api/libraries`.
3. `LibraryStore.load()` silently catches the error (`// Silently fail — individual pages can handle errors`).
4. `loading` flips back to false; `loaded` stays false; `libraries` stays `[]`.
5. Sidebar shows nothing. Libraries page shows "Add A Library" CTA. Dashboard shows "Welcome to Biblioteka".
6. User clicks "Add A Library", creates a duplicate with the same paths.

#### Root cause

`LibraryStore.load()` swallows errors with no `loadError` field. The comment claims "individual pages can handle errors", but `Libraries.svelte:43–47` wraps the call in try/catch expecting the store to throw — and the store never does. Sibling stores (`groups.svelte.ts:17–23`, `reading-lists.svelte.ts:17–20`) correctly expose `loadError` and set `loaded=true` on failure.

#### Evidence

- Buggy: `frontend/src/stores/libraries.svelte.ts:39–51` — `catch { /* silent */ }`.
- UI shows empty state: `frontend/src/components/Libraries.svelte:84–91`.
- Sidebar: `frontend/src/components/Sidebar.svelte:36–39,195`.
- Working pattern: `frontend/src/stores/groups.svelte.ts:17–23`.

#### How to verify

`window.fetch = () => Promise.reject(...)` in dev console; reload the app; observe empty-state CTA on Libraries.

#### Suggested fix

Mirror the `readingListStore` / `groupStore` pattern:

```ts
let loadError = $state<string | null>(null);

async load() {
    if (this.loading) return;
    this.loading = true;
    try {
        this.libraries = await listLibraries();
        this.loaded = true;
        this.loadError = null;
    } catch (e) {
        this.loaded = true;
        this.loadError = getErrorMessage(e);
    } finally {
        this.loading = false;
    }
}
```

Then gate the empty-state UI on `loaded && !loadError`. Render an AlertBanner with a "Retry" button when `loadError` is set.

#### Confidence

- Verified store, UI consumers, and sibling-store comparison.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: DATA_LOSS — user creates duplicate state because UI lied.
- IS_LIVE_SURFACE: every navigation to libraries/dashboard/sidebar.
- NO_SCENARIO_MITIGATION: store swallows; UI can't distinguish.
- CTO_TEST: yes — trust-destroying failure mode.

#### Labels

stale-ui, empty-vs-error, stores, impact:data_loss

---

### Finding 24: SSE error event tells user "metadata fetch failed" while asynq retry is still succeeding

- **Severity**: MEDIUM
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/jobs/enrich_goodreads.go:90–96,119–126`; `frontend/src/components/books/MetadataFetchPanel.svelte:162–166`
- **Trigger condition**: First `enrich:goodreads` attempt fails transiently; asynq retries and succeeds.
- **Consequence**: User sees red "fetch failed" message and cleared spinner. Backend retry succeeds and publishes EventComplete to a channel with no subscribers. User stares at "failed" while pending metadata exists; pending banner only appears after navigate-away-and-back.
- **Verdict**: CONFIRMED

#### What happens

1. User clicks "Fetch Metadata".
2. SSE opens; job enqueued.
3. First attempt: transient Goodreads timeout. Job publishes `EventError` and returns error.
4. Frontend gets `error`, closes SSE, shows "failed to fetch book".
5. Asynq retries (default `MaxRetry=5`). Retry succeeds; publishes `EventComplete` — but no subscriber.
6. DB now has pending metadata. UI shows "fetch failed". User doesn't realize the pending banner is one navigation away.

#### Root cause

Job publishes terminal SSE events on every attempt (treating each failure as final). Frontend treats first `error` as final. Asynq's retry semantics aren't surfaced.

#### Evidence

- Publishes EventError per attempt: `internal/jobs/enrich_goodreads.go:90–96,119–126`.
- Frontend treats first error as final: `frontend/src/components/books/MetadataFetchPanel.svelte:162–166`.
- Default MaxRetry: `internal/worker/worker.go:223`.

#### How to verify

Mock Goodreads to return 500 once then 200. Click Fetch Metadata. Observe "failed" banner; check DB for pending metadata that appeared via the retry.

#### Suggested fix

Two options (pick one):
- (a) Frontend: on EventError, also call `loadPendingMetadata()` — if pending row exists, suppress the error and show the pending banner instead.
- (b) Backend: only publish EventError when asynq exhausts retries (e.g., via `asynq.OnError` hook checking task retry counts). Transient failures shouldn't surface as terminal.

#### Confidence

- Verified job, SSE handler, frontend.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — wrong outcome shown.
- IS_LIVE_SURFACE: primary metadata flow.
- NO_SCENARIO_MITIGATION: frontend has no recovery on error event.
- CTO_TEST: yes — confusing UX on a primary action.

#### Labels

sse, retry, stale-ui, impact:support_burden

---

### Finding 25: Library path edit doesn't trigger a fresh scan; 24h dedup blocks manual re-runs

- **Severity**: MEDIUM
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/handlers/libraries.go:278–312` (update doesn't enqueue); `:220–232` (create uses 24h unique)
- **Trigger condition**: Admin creates a library with a wrong path, edits it to the correct path, saves.
- **Consequence**: No new scan runs. The dedup key uses `library_id` alone; the manually-triggered re-enqueue (if any) within 24h is silently dropped. Admin assumes the scanner is broken.
- **Verdict**: CONFIRMED

#### What happens

1. Admin creates Library "Sci-Fi" → `/wronng/path`. `createLibrary` enqueues `scan:library` with `WithUnique(24h)`.
2. Admin notices typo, edits to `/right/path`. PUT returns 200.
3. `updateLibrary` does NOT enqueue any new scan job.
4. No books appear from `/right/path`. The 24h dedup also blocks any manual re-enqueue keyed on `library_id`.

#### Root cause

`updateLibrary` was never wired to enqueue. Even if it were, the dedup key would need to incorporate path changes.

#### Evidence

- Create enqueue: `internal/handlers/libraries.go:220–232`.
- Update no enqueue: `internal/handlers/libraries.go:278–312`.

#### How to verify

Create a library with a typo'd path; update with the correct path; observe no scan runs.

#### Suggested fix

In `updateLibrary`, after a successful path change, enqueue a fresh `scan:library` job. Compute the unique key from `library_id + paths_hash` so a path change forces a new run:

```go
key := fmt.Sprintf("scan:%s:%s", libraryID, hashPaths(lib.Paths))
h.Enqueuer.Enqueue(ctx, jobs.JobScanLibrary, payload,
    jobs.WithUnique(24*time.Hour),
    asynq.TaskID(key))
```

Optional: expose a "Scan now" button in the UI hitting `POST /api/libraries/{id}/scan`.

#### Confidence

- Verified by reading create and update handlers.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — primary flow has no recovery path.
- IS_LIVE_SURFACE: admin library management.
- NO_SCENARIO_MITIGATION: no enqueue; dedup blocks manual retry.
- CTO_TEST: yes — admin sees "scanner is broken" with no recourse.

#### Labels

scan, dedup, admin, impact:support_burden

---

### Finding 26: Reading-group progress sharing has no per-user opt-in — joining a group exposes progress on every book

- **Severity**: MEDIUM
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/db/reading_groups.go:299–325`; schema `db/migrations/sqlite/20260414000002_create_reading_group_members_table.sql:3`
- **Trigger condition**: User joins any group; other members query `/api/groups/{id}/progress?book_id=X`.
- **Consequence**: Every group member can see this user's `percent_read` and `updated_at` for ANY book — not just the group's shared list. Joining a book club for one title exposes all reading activity to that group.
- **Verdict**: CONFIRMED (validator notes this may be intentional product design — worth product decision)

#### What happens

1. User joins a group.
2. Other members can query progress for any `book_id`.
3. Query LEFT JOINs `kobo_reading_states` for every member with no consent gate.
4. The user's progress on personal/private books is visible to group members.

#### Root cause

`ListGroupMemberProgress` checks `IsMember` then LEFT JOINs `kobo_reading_states` without any consent flag. No `share_progress` column on `reading_group_members`.

#### Evidence

- `internal/db/reading_groups.go:299–325`.
- Schema: `db/migrations/sqlite/20260414000002_create_reading_group_members_table.sql:3`.

#### How to verify

Create a 2-person group. As member A, sync reading progress for an unrelated book. As member B, GET `/api/groups/{groupID}/progress?book_id=<unrelated_book>`. A's progress is exposed.

#### Suggested fix

Add `share_progress BOOLEAN DEFAULT FALSE` to `reading_group_members`. Filter on it in `ListGroupMemberProgress`. Or restrict the endpoint to books in the group's shared lists (`reading_group_lists`).

If product decision is "joining = consent to share all progress", document it in the join-group UX with explicit copy.

#### Confidence

- Verified query and schema.
- Proof script result: N/A.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — privacy expectation gap.
- IS_LIVE_SURFACE: groups feature.
- NO_SCENARIO_MITIGATION: no opt-in column.
- CTO_TEST: depends on product framing; worth a sprint conversation.

#### Labels

privacy, groups, impact:support_burden

---

### Finding 27: SSE write-deadline is silently a no-op — long metadata fetches get torn down at 120 s

- **Severity**: MEDIUM
- **Impact category**: SUPPORT_BURDEN
- **Location**: `internal/handlers/middleware/logging.go:14–50` (statusRecorder missing Unwrap)
- **Trigger condition**: Metadata fetch SSE stream lasts longer than the server's 120 s `HTTPWriteTimeout`.
- **Consequence**: `streamEvents` calls `SetWriteDeadline` to extend deadline beyond `HTTPWriteTimeout`, but the call returns `http.ErrNotSupported` because the response-writer chain breaks at `statusRecorder`. SSE connections die at 120 s regardless of the heartbeat code's intent.
- **Verdict**: CONFIRMED (PROOF PASSED — `/tmp/proof_finding21.go` demonstrated the chain-break end to end)

#### What happens

1. User triggers a long metadata fetch (LLM-driven, 90–180s).
2. `metadata_sse.go:55–60` calls `http.NewResponseController(w).SetWriteDeadline(...)` to push deadline beyond the server's 120 s WriteTimeout.
3. ResponseController walks `Unwrap()` to find the underlying writer that supports `SetWriteDeadline`. The chain is `gzipResponseWriter → statusRecorder → http.response`.
4. `statusRecorder` doesn't implement `Unwrap()`. ResponseController stops there; `statusRecorder` doesn't implement `SetWriteDeadline`; returns `http.ErrNotSupported`.
5. Handler logs a warning and proceeds. SSE connection actually dies at 120 s regardless.

#### Root cause

`statusRecorder` is missing the `Unwrap()` method that `gzipResponseWriter` has. ResponseController can't traverse past it.

#### Evidence

- `internal/handlers/middleware/logging.go:14–50` — no `Unwrap()`.
- `internal/handlers/middleware/gzip.go:71–73` — `gzipResponseWriter.Unwrap()` exists.
- `internal/handlers/metadata_sse.go:55–60` — calls SetWriteDeadline.
- Proof: `/tmp/proof_finding21.go` showed `feature not supported` for both `statusRecorder(real)` and `gzip(statusRecorder(real))`.

#### How to verify

Run `go run /tmp/proof_finding21.go` (from the validation phase). Or simply check `internal/handlers/middleware/logging.go` for any `Unwrap()` method on `statusRecorder` — there isn't one.

#### Suggested fix

One line:

```go
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
```

#### Confidence

- Verified by reading both middlewares.
- Proof script result: PASSED.

#### Gate evidence

- IMPACT_CATEGORY: SUPPORT_BURDEN — primary feature (metadata SSE) degraded silently.
- IS_LIVE_SURFACE: every metadata SSE stream.
- NO_SCENARIO_MITIGATION: warning logged but stream still dies.
- CTO_TEST: yes — silent feature degradation with a one-line fix.

#### Labels

sse, responsecontroller, unwrap, impact:support_burden

---

## Dropped Findings

The following candidates were either DISPROVED, downgraded to LOW (Gate A excludes LOW from the report), or consolidated as duplicates. Listed for transparency:

**Disproved**
- Phase 3 #25 "GetReadingList ignores group sharing" — Reading lists are user-owned by design; group-sharing of list contents is not a documented feature.
- Phase 4 P19 "addGroupMember silent no-op for non-owner returns 204" — The DB layer at `reading_groups.go:202–241` disambiguates owner-vs-already-present and returns `sql.ErrNoRows` for non-owner; the handler maps that to 404, not 204.

**Duplicates (consolidated into the listed finding)**
- Phase 4 P20 "Email send no per-user rate limit" → consolidated into Finding 1.
- Phase 4 P16 "SMTP load failure → password loss" → consolidated into Finding 23-class (dropped: see below; original finding is admin-only and backend mitigates the destructive case).
- Phase 4 P13 "metadataStatusAlreadyRunning UI branch unreachable" → same root cause as Phase 3 #28 (AI enrichment dedup); both dropped to LOW.

**Downgraded to LOW (excluded by Gate A)**
- Phase 3 #11 "Unbounded staged upload accumulation" → LOW-MEDIUM, slow leak not acute.
- Phase 3 #14 "OIDC first-time signup not audit-logged" → LOW.
- Phase 3 #17 "logAudit warning omits userID/action/entityType" → LOW (observability gap, not user-impacting).
- Phase 3 #18 "cover.jpg.tmp overwrite race" → LOW (cosmetic; affects only default `none` org with concurrent uploads to same dir).
- Phase 3 #19 "PUT books no optimistic locking" → LOW (rare in family-deployment trust model).
- Phase 3 #20 "parseLimitOffset offset cap 200K DoS" → LOW (authenticated user in trusted trust boundary).
- Phase 3 #22 "LoggingMiddleware always logs user_id=\"\"" → LOW (audit_logs table is the source of truth).
- Phase 3 #23 "No panic recovery middleware" → LOW (Go stdlib per-conn recover prevents process crash; only structured log/audit lost).
- Phase 3 #24 "POST /api/books/{id}/files duplicate book_file rows" → LOW (migration `20260316000000_add_unique_file_path_index.sql` mitigates the row-duplication; only wrong-status-code issue remains).
- Phase 3 #28 "AI enrichment no dedup on user-triggered path" → LOW (Ollama cost negligible today; narrow race window).
- Phase 4 P3 "AI tag generation unreachable from UI" → LOW (LLM provider is nil in most installs; panel/apply/reject work if a pending row exists).
- Phase 4 P8 "Settings tabs silent load failure" → LOW (admin would refresh on seeing blank fields; backend preserves password for SMTP).
- Phase 4 P11 "Dashboard 'Total Books: 0' on count failure" → LOW (error banner IS shown alongside the zero; misleading polish).
- Phase 4 P12 "Passkey section hidden on getPasskeyEnabled failure" → LOW (transient; reload recovers).
- Phase 4 P15 "groupStore/readingListStore loaded=true on failure" → LOW (intentional pattern; `loadError` IS exposed).
- Phase 4 P17 "No max-length cap on entity names" → LOW (single-tenant trust boundary; one-line fix).
- Phase 4 P18 "Libraries can overlap paths" → LOW (admin-only footgun).
- Phase 4 P21 "Signup email not lowercased" → LOW (DB-side `LOWER()` unique index neutralizes the duplicate-account claim; only casing-display issue remains).
- Phase 4 P22 "Signup accepts malformed emails" → LOW (downstream email features validate at use time).
- Phase 4 P23 "ReadingListDetail blank during initial fetch" → LOW (cosmetic; quickly resolves).
- Phase 4 P24 "Whitespace-only book titles accepted" → LOW (one-line trim fix).
- Phase 4 P25 "POST /books/{id}/files accepts negative size" → LOW (admin-only API).
- Phase 4 P26 "PUT /api/groups/{id} returns 404 to non-owner" → LOW (HTTP-semantic polish).

---

This review ran cold: no team memory, no production signal correlation, no Slack
context, no scheduled cadence, no PR creation, no dedup against existing issues.
[Farfield](https://farfield.dev) runs this same recipe + those five things
continuously, in Slack, against live production telemetry.

If you want findings filed, fixed, and shipped automatically → farfield.dev.
