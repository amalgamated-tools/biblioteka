# OPDS Catalog

Biblioteka includes a built-in [OPDS 1.2](https://specs.opds.io/opds-1.2) catalog server, so any OPDS-compatible e-reader app can browse and download your books directly—no extra software required.

## What Is OPDS?

OPDS (Open Publication Distribution System) is a standard that exposes a library as a browsable Atom feed. Apps such as **KOReader**, **Calibre**, **Moon+ Reader**, **PocketBook**, **Aldiko**, and dozens of others can connect to an OPDS endpoint to search, browse, and download books.

## How OPDS authentication works

The OPDS catalog uses **HTTP Basic Authentication** with a separate username and password — _not_ your Biblioteka account password. This design means you can give the OPDS password to a reading app without exposing your main account credentials.

Each Biblioteka user can set exactly one OPDS credential. The credential is stored as a bcrypt hash; the plaintext password is never persisted.

OPDS credentials are managed through the [JSON API](#managing-opds-credentials-via-the-api), which requires a valid Biblioteka JWT.

### Security note: timing-safe authentication

When a username is not found, the server performs a dummy bcrypt comparison before returning `401 Unauthorized`. This prevents an attacker from enumerating valid usernames via response-time differences.

---

## Catalog URL

```
https://<your-host>/opds
```

All OPDS paths fall under `/opds`. The catalog is scoped to the authenticated user's library.

---

## Setting up OPDS in your reading app

1. **Generate your OPDS credentials** — use the [API](#managing-opds-credentials-via-the-api) to set a username and password for your account.
2. **Add a catalog in your reading app** — enter the URL `https://<your-host>/opds`, and supply the OPDS username and password when prompted.
3. **Browse and download** — your app will display the navigation catalog. Tap a book to see its download links.

> **Username rules:** usernames are case-insensitive and trimmed of whitespace. The username must be unique across all users on the instance.

---

## Catalog structure

| Path | Feed type | Description |
|------|-----------|-------------|
| `/opds` or `/opds/` | Navigation | Root catalog with links to all sections |
| `/opds/all` | Acquisition | All books, paginated |
| `/opds/recent` | Acquisition | Recently added books, paginated |
| `/opds/authors` | Navigation | List of all authors, paginated |
| `/opds/authors/{id}` | Acquisition | Books by a specific author, paginated |
| `/opds/series` | Navigation | List of all series, paginated |
| `/opds/series/{id}` | Acquisition | Books in a specific series, paginated |
| `/opds/search` | OpenSearch description | OpenSearch description document (when no `q` parameter) |
| `/opds/search?q={query}` | Acquisition | Full-text book search results, paginated |
| `/opds/download/{file-id}` | — | Direct file download (streams the file to the client) |

### Page size

All paginated feeds return **50 entries per page**. Use the `?page=N` query parameter to navigate, or follow the `rel="next"` / `rel="previous"` links included in each feed.

### Search

Send a GET request to `/opds/search?q=<query>` to search books by title, author, or description. The search endpoint is also advertised in the root feed as an OpenSearch description document, so compliant clients can discover it automatically.

### File downloads

Each book entry in an acquisition feed contains one `rel="http://opds-spec.org/acquisition"` link per available file format. The link points to `/opds/download/{file-id}`. The server streams the file with a correct `Content-Type` and `Content-Disposition: attachment` header.

Supported MIME types:

| Extension | MIME type |
|-----------|-----------|
| `.epub` | `application/epub+zip` |
| `.pdf` | `application/pdf` |
| `.mobi` | `application/x-mobipocket-ebook` |
| `.azw3` | `application/vnd.amazon.ebook` |
| `.cbz` | `application/vnd.comicbook+zip` |
| `.cbr` | `application/vnd.comicbook-rar` |
| `.fb2` | `application/x-fictionbook+xml` |
| `.txt` | `text/plain` |
| `.djvu` | `image/vnd.djvu` |
| other | `application/octet-stream` |

> **Note on scanner support:** The library scanner automatically imports only `.epub`, `.mobi`, `.azw3`, and `.pdf` files. The additional formats listed above (`.cbz`, `.cbr`, `.fb2`, `.txt`, `.djvu`) are served correctly by the OPDS download endpoint if the corresponding `book_file` records exist in the database, but they are not picked up by the background scanner. To make non-scanned formats available in your catalog, create the book and book_file records manually via the [API](api-reference.md#post-apibooksidfiles-).

### Cover images

When a book has a cover image set, each acquisition feed entry includes a `rel="http://opds-spec.org/image"` link pointing to the cover URL. OPDS clients that support cover art (such as KOReader, Moon+ Reader, and PocketBook) will fetch and display the cover while browsing the catalog.

The `Content-Type` of the cover link is inferred from the image URL's file extension. Biblioteka inspects the trailing portion of the URL, so the URL must literally end with the extension (for example, `https://example.com/covers/1234.png` or `https://example.com/covers/1234.png?size=200` where the path still ends in `.png`). If the extension is unknown, missing, or cannot be cleanly parsed (for example, because query parameters or fragments are included in what looks like the extension), the type falls back to `image/jpeg`.

| Extension | MIME type |
|-----------|-----------|
| `.png` | `image/png` |
| `.webp` | `image/webp` |
| `.avif` | `image/avif` |
| `.gif` | `image/gif` |
| `.svg` | `image/svg+xml` |
| other (including `.jpg` / `.jpeg`, unknown/missing extensions, or URLs where the extension is obscured by query strings/fragments) | `image/jpeg` |

Entries for books without a cover image omit the image link entirely.

---

## Managing OPDS credentials via the API

These endpoints require a valid Biblioteka **JWT** (not the OPDS password). They are documented fully in the [API Reference](api-reference.md#opds-credentials).

### Get current credentials

```http
GET /api/opds/credentials
Authorization: Bearer <jwt>
```

Returns the current OPDS username and timestamps, or `404` if no credentials are configured.

### Create or update credentials

```http
PUT /api/opds/credentials
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "username": "alice",
  "password": "s3cr3t!"
}
```

Creates OPDS credentials if they don't exist, or replaces the current credentials. Returns `409 Conflict` if the username is taken by another user.

### Delete credentials

```http
DELETE /api/opds/credentials
Authorization: Bearer <jwt>
```

Removes the OPDS credential. After deletion, any app using those credentials will receive `401 Unauthorized`.

---

## Error responses

OPDS endpoints always return XML—even for errors—so that OPDS clients receive a parseable Atom feed rather than a JSON error body.

```xml
<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Biblioteka OPDS Error</title>
  <id>urn:biblioteka:opds:error</id>
  <entry>
    <title>Authentication Error</title>
    <content type="text">invalid credentials</content>
  </entry>
</feed>
```

Common status codes for OPDS endpoints:

| Code | Meaning |
|------|---------|
| `401 Unauthorized` | Missing or invalid Basic Auth credentials |
| `404 Not Found` | Author, series, or file not found |
| `405 Method Not Allowed` | Only GET and HEAD are accepted |
| `500 Internal Server Error` | Unexpected server-side failure |

Authentication errors also include a `WWW-Authenticate: Basic realm="Biblioteka OPDS"` response header.

---

## OPDS client examples

### KOReader

1. Open the **OPDS catalog** plugin (*Search → OPDS catalog*).
2. Tap **+** to add a new catalog.
3. Enter the catalog URL: `https://<your-host>/opds`
4. Enter your OPDS username and password.

### Calibre

1. Open *Preferences → Sharing → Content Server* and enable the **Add catalog** button in the Download Books dialog (or use *Connect/Share → Browse by cover*).
2. Alternatively, use Calibre's *Add books → OPDS catalog* importer and enter `https://<your-host>/opds`.
3. Provide your OPDS credentials when prompted.

### Moon+ Reader (Android)

1. Open the app → *Library → Network → OPDS Catalog → Add*.
2. Set the URL to `https://<your-host>/opds`.
3. Enter your OPDS username and password.
