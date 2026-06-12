# Agent Instructions

Biblioteka is a personal digital library management system for cataloging and organizing books. It is a full-stack web application with a Go backend and a Svelte 5 frontend.

NOTE: Use American English spelling in all code, comments, and documentation (e.g., "catalog" not "catalogue").

## Tech Stack

- **Backend**: Go 1.26.2, standard `net/http` (no router framework), `database/sql`
- **Databases**: SQLite (default) and PostgreSQL — both are supported; use dialect-aware helpers
- **Background jobs**: `asynq` (Redis-backed) via `internal/worker`
- **Auth**: [`goauth`](https://github.com/amalgamated-tools/goauth) for JWT, rate limiting, and bcrypt; OIDC via `github.com/coreos/go-oidc/v3`; passkeys via `github.com/go-webauthn/webauthn`
- **Middleware**: `justinas/alice` for middleware chaining
- **Frontend**: Svelte 5, TypeScript, Tailwind CSS 3, Vite
- **Migrations**: `dbmate` format, run automatically on startup

## Project Structure

```
cmd/server/        # Binary entry point
cmd/cli/           # CLI tool for standalone metadata extraction
internal/
  auth/            # Protocol-specific middleware (OPDS, KOSync, Kobo); re-exports JWT, rate limiting, and crypto from goauth
  authstore/       # Adapters bridging db.DB to goauth store interfaces (UserStore, APIKeyStore, PasskeyStore)
  coverutil/       # Cover image decoding (base64 data: URLs; enforces 20 MB limit)
  db/              # Database layer: setup, CRUD per domain (books, authors, …)
  goodreads/       # Goodreads catalog client: search by query/ISBN, lookup by ASIN or Goodreads ID; used by CLI commands
  handlers/        # HTTP handlers, one struct per domain
  handlers/middleware/  # Logging, request ID middleware
  jobs/            # Background job definitions
  metadata/        # EPUB/MOBI/AZW3/PDF metadata extraction via ExifTool
  organize/        # File reorganization into canonical library layouts
  otel/            # OpenTelemetry logging and tracing bootstrap
  otelkeys/        # Predefined slog field-key constants (logger_keys.go)
  pathparser/      # Book path parsing from directory structure
  server/          # HTTP server init, route registration, embedded frontend dist
  sidecar/         # Sidecar file writing: OPF metadata and cover image alongside book files
  telemetry/       # Anonymous usage telemetry (opt-in)
  testutils/       # Test helpers (MakeTestEPUB, MakeTestPDF); used in _test.go files only
  worker/          # asynq worker setup and job handler registration
frontend/src/
  components/      # Svelte page components (PascalCase .svelte files)
  stores/          # Svelte reactive stores (lowercase .ts files)
  lib/api.ts       # Centralised API client
  types/           # Shared TypeScript types (domain-scoped modules)
db/migrations/
  sqlite/          # SQLite migrations (dbmate format)
  postgres/        # PostgreSQL migrations (dbmate format)
```

## OTEL Keys
- Predefine all structured log field keys as constants in `internal/otelkeys/logger_keys.go` (e.g., `UserID`, `BookID`, `RequestID`).
- Use these constants in all logging calls to ensure consistency and enable better log querying.
- If you need a new log field, add a constant in `internal/otelkeys/logger_keys.go` first before using it in your code.
- Keep the keys alphabetized

## After completing a task

- Run `make fmt` and `make hardfmt` before committing.
  - `make fmt` runs `go fmt ./...` (Go) and `pnpm run format` (frontend Prettier)
  - `make hardfmt` runs `go tool gofumpt -w -l .` for strict Go formatting
- Run `pnpm run lint` and `pnpm run check` in `frontend/` before committing frontend code.

## Agentic Workflows

After modifying any `.md` workflow file under `.github/workflows/`, always
recompile and commit the generated workflow files with the source change:

```bash
gh aw compile
apm compile
```

For Goal issues, keep the completion contract evidence-based. A goal is complete
only when the issue's stated verification evidence supports it.

## Commits and pushing

- Do not add a `Co-Authored-By:` trailer to commit messages.
- Require user approval before committing.
- Require user approval before pushing.
- Follow [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/): `<type>[optional scope][optional !]: <description>`
- Common types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`.
- A scope may be added in parentheses (e.g., `fix(parser):`). Use `!` before the colon for breaking changes.
- PR titles must also follow this format since PRs are squash-merged.

## Go Conventions

### Logging

- Always use `log/slog` for structured logging.
- **Always use context-aware variants**: `slog.InfoContext(ctx, ...)`, `slog.ErrorContext(ctx, ...)`, `slog.WarnContext(ctx, ...)`, `slog.DebugContext(ctx, ...)`. The non-context versions (`slog.Info`, `slog.Error`, etc.) are **forbidden by `sloglint`**.
- `log.Print*`, `log.Fatal*`, and `log.Panic*` are **forbidden** by `forbidigo`.
- Pass `r.Context()` in HTTP handlers or propagate `context.Context` through function signatures.
- Do not use raw string keys in log fields; use the predefined constants in `internal/otelkeys/logger_keys.go` (e.g., `otelkeys.UserID`).
- If you need a new log field, add a constant in `internal/otelkeys/logger_keys.go` first.

### Error handling

- Check every error explicitly with `if err != nil`.
- Do not ignore errors in tests either.
- Wrap errors with context: `fmt.Errorf("context: %w", err)`.
- Prefer `errors.Is` for error handling in most cases; only check error strings as a last resort when the error type does not provide enough context.

### HTTP handlers

- Each domain has a handler struct (e.g., `BookHandler`) that holds `*db.DB` and other dependencies.
- Register routes in `internal/server/routes.go` (via `(*Server).setupRoutes`) on the standard `http.ServeMux` — do not introduce a router framework.
- Use `writeJSON(r.Context(), w, status, data)` and `writeError(r.Context(), w, status, message)` from `internal/handlers/response.go` for all responses.
- Extract resource IDs with `extractPathID(r.URL.Path, "/api/books/")` — there are no named URL parameters. To extract a resource ID **and** an optional sub-resource segment, use `extractPathSegments(r.URL.Path, "/api/books/")` which returns `(id, sub, ok)`.
- After fetching a resource by ID, use `handleDBErr(r.Context(), w, err, "book")` to write the error response and return early. It returns `true` when it wrote a response (caller should `return`), `false` when `err == nil`. Maps `sql.ErrNoRows` → `404 Not Found`; all other errors → `500 Internal Server Error`.
- For paginated list endpoints, use `parseLimitOffset(r, defaultPageLimit, maxPageLimit)` from `internal/handlers/pagination.go` to parse `limit` and `offset` query parameters. It silently clamps out-of-range values to safe defaults (`defaultPageLimit = 50`, `maxPageLimit = 200`).
- Before writing a named resource to the database, call `validateName(r.Context(), w, req.Name)` to guard against blank names. It returns `true` when the name is non-blank; on failure it writes a `400 Bad Request` response and returns `false`, so callers can simply return:

  ```go
  if !validateName(r.Context(), w, req.Name) {
      return
  }
  ```

- For named-resource create and update handlers, use `handleNameErr(r.Context(), w, err, db.ErrInvalidXxxName, db.ErrXxxNameExists, "an xxx")` after a failed write to translate sentinel errors into the correct HTTP responses. It returns `true` when it wrote a response (caller should `return`); returns `false` when `err` does not match either sentinel (caller handles the remaining error):

  ```go
  if err := h.DB.CreateAuthor(r.Context(), &author); err != nil {
      if handleNameErr(r.Context(), w, err, db.ErrInvalidAuthorName, db.ErrAuthorNameExists, "an author") {
          return
      }
      slog.ErrorContext(r.Context(), "failed to create author", slog.Any(otelkeys.Error, err))
      writeError(r.Context(), w, http.StatusInternalServerError, "failed to create author")
      return
  }
  ```

`ErrInvalidXxxName` maps to `400 Bad Request` ("name is required"); `ErrXxxNameExists` maps to `409 Conflict` ("an xxx with that name already exists").

- For update handlers, consolidate the full error block with `handleUpdateErr` instead of calling `handleNameErr` and writing a 404 by hand:

  ```go
  if handleUpdateErr(r.Context(), w, err, db.ErrInvalidAuthorName, db.ErrAuthorNameExists, "an author", "author", id) {
      return
  }
  ```

  `handleUpdateErr` returns `true` when it wrote a response (caller should `return`), `false` when `err == nil`. It covers: `sql.ErrNoRows` → `404 Not Found`; `ErrInvalidXxxName` → `400 Bad Request`; `ErrXxxNameExists` → `409 Conflict`; any other error → logs and returns `500 Internal Server Error`.

- For list endpoints that return a slice of DTOs, use the generic `listEntities` helper instead of hand-rolling the list-and-convert pattern:

  ```go
  listEntities(w, r, "authors", h.DB.ListAuthors, toAuthorDTO)
  ```

  `listEntities` is a generic function in `internal/handlers/crud.go`. It calls the `list` function, converts each entity to a DTO via `toDTO`, and writes a `200 OK` JSON response. On error it logs and writes `500 Internal Server Error`. Always `return` immediately after the call.

- When you need to convert a slice of entities to DTOs outside of `listEntities` (for example, in sub-resource handlers whose list function requires additional parameters such as a parent resource ID), use the generic `mapSlice` helper:

  ```go
  writeJSON(ctx, w, http.StatusOK, mapSlice(authors, toAuthorDTO))
  ```

  `mapSlice` is a generic function in `internal/handlers/crud.go`. It applies `toDTO` to every element of `items` and returns the resulting slice. Use it whenever you hold the fetched slice yourself and only need the DTO conversion step (no automatic list call or error handling). The `toDTO` function must accept a **pointer** to the entity type (e.g. `func toAuthorDTO(a *db.Author) AuthorDTO`) — `mapSlice` passes a pointer to each element internally.

- For list endpoints that return a slice of **user-owned** DTOs (where the list function accepts a `userID` as a second argument), use the generic `listUserEntities` helper instead of `listEntities`:

  ```go
  listUserEntities(w, r, "API keys", h.DB.ListAPIKeys, toAPIKeyDTO)
  ```

  `listUserEntities` is a generic function in `internal/handlers/crud.go`. It extracts the authenticated user ID from context via `auth.UserIDFromContext`, calls `list(ctx, userID)`, converts entities to DTOs, and writes a `200 OK` JSON response (never `null`). On error it logs and writes `500 Internal Server Error`. Always `return` immediately after the call.

- For **paginated** list endpoints that return a list DTO wrapping the items together with `total`, `limit`, and `offset` fields (e.g., `authorListDTO`, `seriesListDTO`, `tagListDTO`), use the generic `listPaginatedEntities` helper instead of hand-rolling the limit/offset parsing and list-DTO assembly:

  ```go
  listPaginatedEntities(w, r, "authors", h.DB.ListAuthorsPaginated, toAuthorDTO,
      func(items []authorDTO, total, limit, offset int) authorListDTO {
          return authorListDTO{
              Authors: items,
              Total:   total,
              Limit:   limit,
              Offset:  offset,
          }
      },
  )
  ```

  `listPaginatedEntities` is a generic function in `internal/handlers/crud.go`. It calls `parseLimitOffset(r, defaultPageLimit, maxPageLimit)` to parse and clamp the `limit`/`offset` query parameters, invokes the paginated `list(ctx, limit, offset) ([]T, int, error)` function, converts each entity to a DTO via `toDTO`, and writes a `200 OK` JSON response wrapping the result through `toListDTO(items, total, limit, offset)`, where `items` is the converted `[]DTO` slice (not the raw entities). On error it logs and writes `500 Internal Server Error`. Always `return` immediately after the call. Use this helper for top-level paginated endpoints such as `GET /api/authors`, `GET /api/series`, and `GET /api/tags`.

### Book sub-resource handlers

For GET and PUT handlers that operate on a book's associated sub-resources (such as authors or series linked to a book), use `respondBookSubResource` and `putBookSubResource` from `internal/handlers/book_subresource.go` instead of hand-rolling the fetch → convert → respond pattern:

```go
// GET handler
func (h *BookHandler) handleGetBookAuthors(w http.ResponseWriter, r *http.Request, bookID string) {
    respondBookSubResource(r.Context(), w, bookID, h.DB.GetBookAuthors, toAuthorDTO, "book authors")
}

// PUT handler
func (h *BookHandler) handleSetBookAuthors(w http.ResponseWriter, r *http.Request, bookID string) {
    putBookSubResource(w, r, bookID,
        h.DB.GetBookAuthors, h.DB.SetBookAuthors,
        func(req *setBookAuthorsRequest) []string { return req.AuthorIDs },
        toAuthorDTO,
        "book authors",
    )
}
```

- `respondBookSubResource[T, DTO](ctx, w, bookID, getFn, toDTO, resourceName)` — calls `getFn(ctx, bookID)`, converts each element via `mapSlice(items, toDTO)`, and writes a `200 OK` JSON response. On error it logs and writes `500 Internal Server Error`.
- `putBookSubResource[T, DTO, Req, Payload](w, r, bookID, getFn, setFn, extractPayload, toDTO, resourceName)` — decodes the JSON request body into `Req`, extracts the payload via `extractPayload`, calls `setFn` to persist the change, then delegates to `respondBookSubResource` to re-fetch and return the updated resource. On decode or set error it writes the appropriate error response.

Both helpers are unexported and live in `internal/handlers/book_subresource.go`. Use them whenever you add a new relationship endpoint (GET or PUT) that replaces a book's full set of associated entities (e.g., authors, series, tags).

### Named-entity CRUD handlers

When adding a new entity type that has a name, a GET-by-ID, a create, and an update endpoint (e.g., an author or a series), use `namedEntityOps` and its three generic helpers from `internal/handlers/named_entity.go` instead of hand-rolling the decode → validate → write → audit flow for each operation:

```go
func (h *TagHandler) tagOps() namedEntityOps[db.Tag, tagDTO, tagRequest] {
    return namedEntityOps[db.Tag, tagDTO, tagRequest]{
        db:             h.DB,
        entityLabel:    "tag",
        entityArticle:  "a tag",
        idKey:          otelkeys.TagID,
        errInvalidName: db.ErrInvalidTagName,
        errNameExists:  db.ErrTagNameExists,
        auditCreate:    db.AuditActionTagCreated,
        auditUpdate:    db.AuditActionTagUpdated,
        get:            h.DB.GetTag,
        create:         func(ctx context.Context, req tagRequest) (*db.Tag, error) { return h.DB.CreateTag(ctx, req.Name) },
        update:         func(ctx context.Context, id string, req tagRequest) (*db.Tag, error) { return h.DB.UpdateTag(ctx, id, req.Name) },
        reqName:        func(req tagRequest) string { return req.Name },
        entityName:     func(t *db.Tag) string { return t.Name },
        entityID:       func(t *db.Tag) string { return t.ID },
        toDTO:          toTagDTO,
    }
}

func (h *TagHandler) HandleTag(w http.ResponseWriter, r *http.Request) {
    id, ok := extractPathID(r.URL.Path, "/api/tags/")
    if !ok {
        writeError(r.Context(), w, http.StatusBadRequest, "invalid tag ID")
        return
    }
    switch r.Method {
    case http.MethodGet:
        getNamedEntity(h.tagOps(), w, r, id)
    case http.MethodPut:
        updateNamedEntity(h.tagOps(), w, r, id)
    default:
        writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
    }
}
```

- `createNamedEntity(ops, w, r)` — decode request → validate name → call `ops.create` → handle name errors → audit → `201 Created`
- `getNamedEntity(ops, w, r, id)` — call `ops.get` → handle DB errors → `200 OK`
- `updateNamedEntity(ops, w, r, id)` — decode request → validate name → call `ops.update` → handle name/not-found errors → audit → `200 OK`

For list and delete, continue using `listEntities` (or `listUserEntities`) and `deleteResource` directly — `namedEntityOps` covers only the create, get, and update flows.

### Deleting a resource

For DELETE handlers, use the generic `deleteResource` helper instead of hand-rolling the fetch-delete-audit pattern:

```go
deleteResource(h.DB, w, r, id, "author", "author", otelkeys.AuthorID,
    h.DB.GetAuthor, h.DB.DeleteAuthor,
    db.AuditActionAuthorDeleted,
    func(a *db.Author) map[string]any { return map[string]any{"name": a.Name} },
)
```

`deleteResource` is a package-level generic function in `internal/handlers/crud.go`. It fetches the entity (to capture audit metadata), deletes it, writes an audit log entry via `db.CreateAuditLog`, and responds with `204 No Content`. A failed audit write is logged as a warning and never blocks the response. Pass `nil` for `auditMeta` when no extra metadata is needed. Always `return` immediately after the call — `deleteResource` always writes the HTTP response itself.

### Deleting a user-owned resource

For user-owned resources (such as API keys and Kobo tokens) where the get and delete functions accept both a resource ID and a user ID, use `deleteUserOwnedResource` instead:

```go
deleteUserOwnedResource(h.DB, w, r, id, "API key", "api_key", otelkeys.APIKeyID,
    h.DB.GetAPIKey, h.DB.DeleteAPIKey,
    db.AuditActionAPIKeyDeleted,
    func(k *db.APIKey) map[string]any { return map[string]any{"name": k.Name} },
)
```

`deleteUserOwnedResource` mirrors `deleteResource` in behavior — it fetches the entity, deletes it, writes an audit log entry, and responds with `204 No Content`. Pass the human-readable display name as `resource` (e.g. `"API key"`) and the stable snake_case identifier as `auditEntityType` (e.g. `"api_key"`). Pass `nil` for `auditMeta` when no extra metadata is needed. The user ID is extracted from context automatically via `auth.UserIDFromContext`. Always `return` immediately after the call.

### Creating user-owned tokens

When adding a new token type (a high-entropy random secret stored by hash, such as an API key or Kobo sync token), use `tokenOps` and `handleTokenCreate` from `internal/handlers/tokens_compat.go` instead of hand-rolling the decode → validate → generate → hash → persist → audit flow. Both are unexported, so new token handlers must live in the `internal/handlers` package:

```go
func (h *MyTokenHandler) createMyToken(w http.ResponseWriter, r *http.Request) {
    handleTokenCreate(tokenOps{
        db:              h.DB,
        resource:        "my token",
        auditEntityType: "my_token",
        auditCreate:     db.AuditActionMyTokenCreated,
        create: func(ctx context.Context, userID, name string) (string, any, error) {
            raw, err := generateRandomHex(32) // 64-char hex token
            if err != nil {
                return "", nil, &tokenError{err: err, message: "failed to generate my token"}
            }
            hash := auth.HashMyToken(raw)
            token, err := h.DB.CreateMyToken(ctx, userID, name, hash)
            if err != nil {
                return "", nil, err
            }
            return token.ID, myTokenCreateResponse{myTokenDTO: toMyTokenDTO(token), Token: raw}, nil
        },
    }, w, r)
}
```

`handleTokenCreate` implements the full creation lifecycle: decode the `{"name": "..."}` request body, validate and trim the name (≤ 100 characters; see `maxTokenNameLength`), call `ops.create`, write an audit log entry, and respond with `201 Created` via `writeSecretTokenResponse` (which sets `Cache-Control: no-store` and `Pragma: no-cache` to prevent caching of the plaintext secret). The raw token is returned only in the creation response and cannot be retrieved again.

Use `generateRandomHex(n)` from `internal/handlers/auth_compat.go` to generate a cryptographically secure random token of `n` bytes (returned as a `2n`-character lowercase hex string).

### Audit logging (non-`deleteResource` actions)

For actions not covered by `deleteResource`, call `logAudit` after the database write succeeds:

```go
logAudit(r.Context(), h.DB, userID, db.AuditActionBookCreated, "book", b.ID, map[string]any{"title": b.Title})
```

`logAudit` is a package-level function in `internal/handlers/dberrors.go`. It calls `db.CreateAuditLog` and logs a warning on failure without propagating the error, so a failed audit write never causes a request to fail. The caller must supply `userID`, typically obtained via `auth.UserIDFromContext(r.Context())`.

### Admin protection

```go
if !requireAdmin(h.DB, w, r) {
    return
}
```

`requireAdmin` is a package-level function in `internal/handlers/crud.go`. It writes the error response itself; return immediately when it returns `false`.

### Protocol credential handlers

When adding a new protocol that requires user-set credentials (username + bcrypt-hashed password), use `credentialOps` and `handleCredentials` from `internal/handlers/credentials.go` instead of hand-rolling GET/PUT/DELETE logic. Both are unexported, so new protocol handlers must live in the `internal/handlers` package:

```go
func (h *MyProtocolHandler) HandleMyProtocolCredentials(w http.ResponseWriter, r *http.Request) {
    handleCredentials(credentialOps{
        db:              h.DB,
        protocol:        "MyProtocol",
        auditEntityType: "myprotocol_credential",
        auditUpsert:     db.AuditActionMyProtocolCredentialUpdated,
        auditDelete:     db.AuditActionMyProtocolCredentialDeleted,
        errConflict:     db.ErrMyProtocolUsernameExists,
        getByUserID:     credGetAdapter(h.DB.GetMyProtocolCredentialByUserID),
        upsert:          credUpsertAdapter(h.DB.UpsertMyProtocolCredential),
        del:             h.DB.DeleteMyProtocolCredential,
    }, w, r)
    // NOTE: handleCredentials writes the HTTP response; always return immediately after calling it.
    return
}
```

`handleCredentials` dispatches GET, PUT, and DELETE to the appropriate inner function. It handles:

- Username normalization (lowercase, trimmed) and length validation (≤ 256 bytes; see `maxUsernameLen`)
- Password validation and bcrypt hashing
- Upsert semantics with username-conflict detection (`errConflict` → `409 Conflict`)
- Audit logging for both upsert and delete actions
- Structured error responses for all failure cases

Set `deriveKey` when the protocol requires a password transformation before hashing (KOSync uses MD5 to match the KOReader protocol specification). Leave `deriveKey` as `nil` to hash the plaintext password directly (OPDS).

Use `credGetAdapter` and `credUpsertAdapter` (generic functions in `internal/handlers/credentials.go`) to adapt your DB methods for the `credentialOps` struct. They accept any DB method whose return type satisfies `credentialInfoer` (the interface implemented by `db.ProtocolCredential` and all type aliases derived from it, such as `db.OPDSCredential` and `db.KOSyncCredential`) and wrap it in the closure signature that `credentialOps.getByUserID` and `credentialOps.upsert` require:

- `credGetAdapter(fn)` — wraps a `func(context.Context, userID string) (T, error)` DB getter
- `credUpsertAdapter(fn)` — wraps a `func(context.Context, userID, username, hash string) (T, error)` DB upsert

### Protocol authentication middleware

When adding a new protocol that authenticates incoming requests with a bcrypt-hashed credential (e.g., a new sync protocol using custom headers or Basic Auth), use `bcryptCredMiddleware` from `internal/auth/credential_middleware.go` instead of hand-rolling the extract → lookup → compare → inject flow.

Because `bcryptCredMiddleware` is unexported and lives in `internal/auth`, any new protocol auth middleware that uses it must also be implemented in the `internal/auth` package (similar to how unexported helpers require handlers to live in `internal/handlers`).
```go
func MyProtocolAuthMiddleware(checker MyProtocolCredentialChecker) func(http.Handler) http.Handler {
    return bcryptCredMiddleware(bcryptCredConfig{
        protocolName: "MyProtocol",
        dummyHash:    mustGenerateDummyBcryptHash("dummy-myprotocol-password", "MyProtocol"),
        usernameAttr: func(v string) slog.Attr { return slog.String(otelkeys.MyProtocolUsername, v) },
        extractCreds: func(r *http.Request) (username, secret string, ok bool) {
            // pull credentials from headers, Basic Auth, etc.
        },
        lookupCredential: func(ctx context.Context, username string) (string, string, error) {
            cred, err := checker.GetMyProtocolCredential(ctx, username)
            if err != nil {
                return "", "", err // must be (wrapped) sql.ErrNoRows for "user not found"
            }
            return cred.UserID, cred.PasswordHash, nil
        },
        writeMissing: func(w http.ResponseWriter, r *http.Request) {
            // write 401 when credentials are absent from the request
        },
        writeUnauthorized: func(w http.ResponseWriter, r *http.Request) {
            // write 401 when username is unknown or password is wrong
        },
        // writeServiceUnavailable is optional; omit to treat all lookup errors as 401
    })
}
```

`bcryptCredMiddleware` is an unexported function in `internal/auth/` that:

- Extracts credentials via `extractCreds`; calls `writeMissing` and stops when none are present
- Normalizes the username (lowercase, trimmed) before calling `lookupCredential`
- Performs a constant-time dummy bcrypt comparison when the username is not found, preventing timing-based username enumeration attacks
- Calls `bcrypt.CompareHashAndPassword` to verify the credential; calls `writeUnauthorized` on failure
- On success, injects the authenticated `userID` into `r.Context()` and calls the next handler

Set `writeServiceUnavailable` to surface transient credential-lookup errors as `503 Service Unavailable`; omit it to silently treat all lookup failures as an unknown username (OPDS behavior). Always add a pre-computed `dummyHash` via `mustGenerateDummyBcryptHash` to ensure the dummy comparison uses a valid hash cost.

### User data isolation

Every database query that reads or writes user-owned data **must** filter by `user_id`. Never return data across users.

### Dependencies

- Avoid adding new dependencies. Discuss in an issue first — the project values minimal dependencies.
- Never edit `*.gen.go` files by hand; regenerate with `go generate ./...`.

## Database Conventions

- Migrations live in `db/migrations/sqlite/` and `db/migrations/postgres/`.
- Name files with a timestamp prefix: `YYYYMMDDHHMMSS_description.sql`.
- Use dbmate format:
  ```sql
  -- migrate:up
  CREATE TABLE ...;

  -- migrate:down
  DROP TABLE ...;
  ```
- Migrations run automatically on server startup.
- SQLite connections use `PRAGMA journal_mode=WAL`, `synchronous=NORMAL`, and `foreign_keys=ON`.
- Use `db.Timestamp` for time columns and `db.now()` for dialect-aware current-time expressions.

### FindOrCreate pattern

When implementing a `FindOrCreate*` function for a new named entity (such as a tag or publisher), use the unexported `findOrCreate` generic helper from `internal/db/find_or_create.go` instead of re-implementing the lookup → insert → race-fetch sequence:

```go
func (d *DB) FindOrCreateTag(ctx context.Context, name string) (*Tag, error) {
    return findOrCreate(ctx, name, "tag",
        NormalizeTagName, ErrInvalidTagName, ErrTagNameExists,
        d.GetTagByName,
        func(ctx context.Context, n string) (*Tag, error) {
            return d.CreateTag(ctx, n)
        },
    )
}
```

`findOrCreate` normalizes the name, validates it against `errInvalid`, handles concurrent-insert races (unique-constraint violation → retry fetch), and emits a debug log. Pass the raw (un-normalized) name — normalization is performed inside the helper.

### Named-entity create and update (db layer)

When implementing `Create*` and `Update*` functions for a named entity, use the unexported `namedEntityCreate` and `namedEntityUpdate` generic helpers from `internal/db/named_entity_write.go` instead of re-implementing the normalize → validate → write → translate-constraint pattern by hand:

```go
func (d *DB) CreateTag(ctx context.Context, name string) (*Tag, error) {
    return namedEntityCreate(ctx, "tag", name,
        NormalizeTagName, ErrInvalidTagName, ErrTagNameExists,
        func(ctx context.Context, n string) (*Tag, error) {
            // execute the INSERT and scan the result
        },
    )
}

func (d *DB) UpdateTag(ctx context.Context, id, name string) (*Tag, error) {
    return namedEntityUpdate(ctx, "tag", id, name,
        NormalizeTagName, ErrInvalidTagName, ErrTagNameExists,
        func(ctx context.Context, id, n string) (*Tag, error) {
            // execute the UPDATE and scan the result
        },
    )
}
```

Both helpers normalize the name via `normalize`, reject a blank result with `errInvalid`, execute the provided insert/update function, and translate a unique-constraint violation into `errExists`. A warn-level log is emitted on a blank name; a debug-level log is emitted before the write. Pass the raw (un-normalized) name — normalization is performed inside the helper.

### Protocol credential database layer

When implementing the database layer for a new sync protocol that stores a username + bcrypt-hashed password, use the shared helpers in `internal/db/protocol_credentials.go` instead of hand-rolling the SQL. Define a `protocolCredentialConfig` value, declare a type alias for `ProtocolCredential`, and delegate all CRUD to the unexported package-level helpers:

```go
// ErrMyProtocolUsernameExists is returned when a MyProtocol username is already taken by another user.
var ErrMyProtocolUsernameExists = errors.New("myprotocol username already exists")

// MyProtocolCredential represents a row in the myprotocol_credentials table.
type MyProtocolCredential = ProtocolCredential

var myprotocolCredConfig = protocolCredentialConfig{
    table:        "myprotocol_credentials",
    tableCol:     "myprotocol_credentials.username",
    indexName:    "idx_myprotocol_credentials_username",
    errExists:    ErrMyProtocolUsernameExists,
    logPrefix:    "MyProtocol",
    usernameAttr: func(v string) slog.Attr { return slog.String("myprotocol.username", v) },
}

func (d *DB) GetMyProtocolCredentialByUserID(ctx context.Context, userID string) (*MyProtocolCredential, error) {
    return getCredentialByUserID(ctx, d, myprotocolCredConfig, userID)
}

func (d *DB) GetMyProtocolCredentialByUsername(ctx context.Context, username string) (*MyProtocolCredential, error) {
    return getCredentialByUsername(ctx, d, myprotocolCredConfig, username)
}

// UpsertMyProtocolCredential creates or updates the MyProtocol credential for a user.
// Returns ErrMyProtocolUsernameExists if the username is taken by a different user.
func (d *DB) UpsertMyProtocolCredential(ctx context.Context, userID, username, passwordHash string) (*MyProtocolCredential, error) {
    return upsertCredential(ctx, d, myprotocolCredConfig, userID, username, passwordHash)
}

func (d *DB) DeleteMyProtocolCredential(ctx context.Context, userID string) error {
    return deleteCredential(ctx, d, myprotocolCredConfig, userID)
}
```

`protocolCredentialConfig` holds the table name and the unique-constraint identifiers used to detect username conflicts. `upsertCredential` performs the `ON CONFLICT (user_id) DO UPDATE` upsert and translates a unique-constraint violation on the `username` column into `errExists` — which `credentialOps.errConflict` then maps to a `409 Conflict` response in the handler. SQLite may reference either the fully qualified column (`tableCol`) or the index name (`indexName`) in its error message; PostgreSQL references the index/constraint name — hence the `protocolCredentialConfig` accepts both.

`ProtocolCredential` (defined in `protocol_credentials.go`) is the shared base struct. Declare your protocol-specific type as a type alias (`type MyProtocolCredential = ProtocolCredential`) so callers get a named type without duplicating fields.

## Frontend Conventions

- Use TypeScript strict mode; put shared types in domain-scoped modules under `frontend/src/types/` (e.g. `book.ts`, `auth.ts`); re-export everything via `frontend/src/types/index.ts`.
- All API calls go through `src/lib/api.ts`.
- Manage reactive state in Svelte stores under `src/stores/`.
- Style with Tailwind CSS utility classes — no component library.
- Component files are PascalCase `.svelte`; store files are lowercase `.ts`.
- Run `pnpm run format` (Prettier), `pnpm run lint` (ESLint), and `pnpm run check` (svelte-check) before committing frontend changes.

## Testing

```bash
# Go tests
go test ./...

# Frontend tests
cd frontend && pnpm run test
```

- Go tests use a real SQLite database configured with WAL, `synchronous=NORMAL`, and `foreign_keys=ON` (see `internal/db/testhelper_test.go`).
- **Use `testify/require`** (e.g., `require.NoError`, `require.Equal`) for test assertions instead of `t.Fatal`, `t.Fatalf`, or `t.FailNow`. This keeps assertion style consistent across the codebase.
- **Every new feature, component, handler, or function must include tests.** This applies to both Go and frontend code. Do not consider a task complete until tests are written and passing.

## Common Commands

```bash
make dev        # Start backend (air hot-reload) + Vite frontend dev server
make build      # Build frontend then compile Go binary
make fmt        # Format Go (go fmt ./...) and frontend (pnpm run format / Prettier)
make hardfmt    # Strict Go formatting (go tool gofumpt -w -l .)
go test ./...   # Run all Go tests
cd frontend && pnpm run lint && pnpm run check   # Lint & type-check frontend
```

## graphify

This project has a graphify knowledge graph at docs/graph/.

Rules:
- Before answering architecture or codebase questions, read docs/graph/GRAPH_REPORT.md for god nodes and community structure
- If docs/graph/wiki/index.md exists, navigate it instead of reading raw files
- After modifying code files in this session, run `python3 -c "from graphify.watch import _rebuild_code; from pathlib import Path; _rebuild_code(Path('.'))"` to keep the graph current
