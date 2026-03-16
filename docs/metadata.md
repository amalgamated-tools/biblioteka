# Metadata Extraction

Biblioteka can extract metadata — title, author, ISBN, format — from book files using two complementary code paths:

| Path | Formats | External dependency |
|------|---------|---------------------|
| **Native EPUB** | `.epub` | None |
| **ExifTool** | `.mobi`, `.azw3`, `.pdf` | [ExifTool](https://exiftool.org/) must be on `PATH` |

The extractor is implemented in [`internal/metadata/extractor.go`](../internal/metadata/extractor.go) and exposed to end users via the standalone [`cmd/cli`](../cmd/cli/main.go) utility.

> **Import pipeline status:** Automatic metadata extraction is **now active** in the `process:file` background job (since v0.0.5). When a file is imported during a library scan, the extractor runs and populates the book record with `Title`, `ISBN` (stored as ISBN-10 or ISBN-13), `Description`, and `Publisher` (EPUB only) when the extracted value is non-empty. If extraction fails (for example, because ExifTool is not installed), the job falls back to deriving the book title from the filename. Author records are also created automatically from extracted author metadata and linked to the imported book.

---

## Extracted fields

| Field | EPUB (native) | MOBI / AZW3 / PDF (ExifTool) | Notes |
|-------|:---:|:---:|-------|
| `Title` | ✓ | ✓ | Falls back to filename (without extension) when the ExifTool path cannot find a `Title` tag |
| `Author` | ✓ | ✓ | Returns `""` when not found; no author record is created for an empty author |
| `ISBN` | ✓ | ✓ | Returns `""` when no valid 10- or 13-digit identifier is present; MOBI files also try an `Identifier` tag as a fallback |
| `Format` | ✓ | ✓ | Uppercase file extension (e.g. `"EPUB"`, `"PDF"`) |
| `IsNative` | `true` | `false` | Indicates whether the native EPUB parser was used |
| `Publisher` | ✓ | ✗ | Extracted from `<dc:publisher>` in EPUB OPF; not available for ExifTool-based formats |
| `Description` | ✓ | ✗ | Extracted from `<dc:description>` in EPUB OPF; not available for ExifTool-based formats |
| `Language` | ✓ | ✗ | Extracted from `<dc:language>` in EPUB OPF; not available for ExifTool-based formats |
| `PublicationDate` | ✓ | ✗ | Extracted from the `<dc:date event="publication">` element in EPUB OPF; not available for ExifTool-based formats |

---

## EPUB extraction (native)

EPUB files are parsed directly using the [`goreader/epub`](https://github.com/taylorskalyo/goreader) library — no external process is required.

The extractor reads the first OPF rootfile inside the ZIP container and maps:

| OPF field | `BookMetadata` field |
|-----------|---------------------|
| `<dc:title>` | `Title` |
| `<dc:creator>` | `Author` |
| `<dc:publisher>` | `Publisher` |
| `<dc:description>` | `Description` |
| `<dc:language>` | `Language` |
| `<dc:date event="publication">` | `PublicationDate` |
| `<dc:identifier>` | `ISBN` (cleaned of `urn:isbn:` / `isbn:` prefixes; validated as 10 or 13 digits) |

---

## ExifTool extraction (MOBI, AZW3, PDF)

Non-EPUB formats are handled by spawning [ExifTool](https://exiftool.org/) as a subprocess and reading the returned tag map.

Tag mapping:

| ExifTool tag | `BookMetadata` field | Fallback |
|--------------|---------------------|----------|
| `Title` | `Title` | Filename stem |
| `Author` | `Author` | `""` (empty; no author record created) |
| `ISBN` | `ISBN` | `Identifier` tag, then `""` (empty) |

When ExifTool is **not installed**, `NewExtractor()` still returns a valid `*Extractor` (with a warning logged), but calling `ExtractMetadata` on a non-EPUB file returns an error:

```
exiftool is not available on this system
```

EPUB extraction continues to work without ExifTool.

### Installing ExifTool

| Platform | Command |
|----------|---------|
| Ubuntu / Debian | `apt-get install libimage-exiftool-perl` |
| macOS | `brew install exiftool` |
| Alpine | `apk add exiftool` |
| Windows | Download from [exiftool.org](https://exiftool.org/) and add to `PATH` |

The Dockerfiles included in this repository already install ExifTool.

---

## CLI tool

`cmd/cli` is a standalone wrapper around the book-processing pipeline, useful for importing a single file and inspecting the result outside of the server. It extracts metadata, stores a book and book_file record in the configured database, and creates an author record if one is found.

```bash
# Build
go build -o biblioteka-cli ./cmd/cli

# Usage
./biblioteka-cli /path/to/book.epub
./biblioteka-cli /path/to/book.mobi
./biblioteka-cli /path/to/book.pdf
```

**Example output (successful import):**

```
Successfully processed file: /path/to/book.epub
```

**Example output (PDF without ExifTool):**

```
Successfully processed file: /path/to/book.pdf
```

> **Note:** When ExifTool is not installed, PDF and MOBI/AZW3 imports still succeed. The book title falls back to the filename (without extension), and no author, ISBN, description, or other metadata is populated. Install ExifTool to enable richer metadata extraction for these formats.

> **Note:** The CLI requires a database to be configured via the same environment variables as the server (see [deployment.md](deployment.md)). It inserts records directly into the database rather than going through the background job queue.

---

## What's next

The `process:file` background job ([`internal/jobs/process_file.go`](../internal/jobs/process_file.go)) extracts and stores `Title`, `ISBN`, `Description`, `Publisher`, `Language`, `PublicationDate` (EPUB only), and links extracted `Author` names to book records. Planned future improvements include:

1. **Publisher, Language, and PublicationDate for ExifTool formats** — extract and store these fields for MOBI, AZW3, and PDF files.
2. **Cover image** — populate `cover_image_url` for formats that embed cover art.

Use `cmd/cli` to import a single file and verify what Biblioteka extracts before a full library scan.

---

## Contributing

To add support for a new field or format:

1. Edit [`internal/metadata/extractor.go`](../internal/metadata/extractor.go).
2. For EPUB: extend `extractNativeEpub` to read the relevant OPF element.
3. For ExifTool-based formats: add the appropriate `book.GetString("<TagName>")` call in `extractExif`.
4. Add or extend tests in `internal/metadata/extractor_test.go`.
5. Update the [Extracted fields](#extracted-fields) table in this document.
