# Biblioteka

A self-hosted personal book library manager. Scan local files, extract metadata, and browse your e-book and physical book collection through a clean web interface.

## Features

- **Multi-format support** — EPUB, MOBI, AZW3, and PDF
- **Automatic metadata extraction** — title, author, ISBN, description, publisher, language, and publication date via [ExifTool](https://exiftool.org/)
- **Path-based metadata** — derives author, title, series name, and position from directory structure
- **File organisation** — configurable layouts: `book_per_folder`, `book_per_file`, or `none`
- **Sidecar files** — writes OPF metadata and cover images alongside book files (Calibre/KOReader/Kobo compatible)
- **Multiple libraries** — group books into named libraries with configurable paths
- **Author and series tracking** — browse by author or series with position numbers
- **User authentication** — JWT-based login with optional OIDC/SSO
- **API keys** — long-lived tokens for programmatic access
- **OPDS 1.2 catalog** — browse and download books from any OPDS-compatible e-reader app
- **Kobo e-reader sync** — native Kobo device API for library and reading progress sync
- **KOReader sync** — kosync-compatible API for reading position sync
- **Background processing** — Redis-backed job queue with built-in monitoring UI
- **Two database backends** — SQLite (zero-config default) or PostgreSQL
- **Single binary** — Go backend embeds the Svelte frontend

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go, `net/http` |
| Frontend | Svelte 5, TypeScript, Tailwind CSS, Vite |
| Database | SQLite (default) or PostgreSQL |
| Job queue | asynq (Redis) |
| Auth | JWT and OIDC |
| Observability | OpenTelemetry (tracing + structured logging) |

## Quick Links

- [Deployment](deployment.md) — get Biblioteka running with Docker or from source
- [Authentication](authentication.md) — configure JWT, OIDC, and account linking
- [Administration](administration.md) — manage users, libraries, and file organization
- [API Reference](api-reference.md) — complete REST API documentation
- [OPDS Catalog](opds.md) — set up e-reader access
- [Kobo Sync](kobo.md) — sync with Kobo e-readers
- [KOReader Sync](koreader.md) — sync reading progress with KOReader
