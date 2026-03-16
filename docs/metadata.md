# Metadata Extraction

Biblioteka can extract metadata — title, author, ISBN, format — from book files using two complementary code paths:

| Path | Formats | External dependency |
|------|---------|---------------------|
| **Native EPUB** | `.epub` | None |
| **ExifTool** | `.mobi`, `.azw3`, `.pdf` | [ExifTool](https://exiftool.org/) must be on `PATH` |

The extractor is implemented in [`internal/metadata/extractor.go`](../internal/metadata/extractor.go) and exposed to end users via the standalone [`cmd/cli`](../cmd/cli/main.go) utility.

> **Import pipeline status:** Automatic metadata extraction is **now active** in the `process:file` background job (since v0.0.5). When a file is imported during a library scan, the extractor runs and populates the book record with `Title`, `ISBN` (stored as ISBN-10 or ISBN-13), `Description`, `Publisher`, `Language`, and `PublicationDate` when the extracted values are non-empty. An `Author` record is also created (or looked up) and linked to the book when an author name is found. If extraction fails (for example, because ExifTool is not installed), the job falls back to deriving the book title from the filename.

---

## Extracted fields

| Field | EPUB (native) | MOBI / AZW3 / PDF (ExifTool) | Notes |
|-------|:---:|:---:|-------|
| `Title` | ✓ | ✓ | Falls back to filename (without extension) when the ExifTool path cannot find a `Title` tag |
| `Author` | ✓ | ✓ | Empty string when no author tag is found; ExifTool falls back to `""` |
| `ISBN` | ✓ | ✓ | Empty string when no valid 10- or 13-digit identifier is present; MOBI files also try an `Identifier` tag as a fallback |
| `Format` | ✓ | ✓ | Uppercase file extension (e.g. `"EPUB"`, `"PDF"`) |
| `IsNative` | `true` | `false` | Indicates whether the native EPUB parser was used |
| `Description` | ✓ | ✗ | Populated from `<dc:description>` in EPUB OPF; not yet extracted for ExifTool-based formats |
| `Publisher` | ✓ | ✗ | Populated from `<dc:publisher>` in EPUB OPF; not yet extracted for ExifTool-based formats |
| `Language` | ✓ | ✗ | Populated from `<dc:language>` in EPUB OPF; not yet extracted for ExifTool-based formats |
| `PublicationDate` | ✓ | ✗ | Populated from the `<dc:date event="publication">` element in EPUB OPF; not yet extracted for ExifTool-based formats |

---

## EPUB extraction (native)

EPUB files are parsed directly using the [`goreader/epub`](https://github.com/taylorskalyo/goreader) library — no external process is required.

The extractor reads the first OPF rootfile inside the ZIP container and maps:

| OPF field | `BookMetadata` field | Notes |
|-----------|---------------------|-------|
| `<dc:title>` | `Title` | |
| `<dc:creator>` | `Author` | |
| `<dc:identifier>` | `ISBN` | Cleaned of `urn:isbn:` / `isbn:` prefixes; validated as 10 or 13 digits |
| `<dc:description>` | `Description` | |
| `<dc:publisher>` | `Publisher` | |
| `<dc:language>` | `Language` | |
| `<dc:date event="publication">` | `PublicationDate` | Only the element with `event="publication"` is used |

---

## ExifTool extraction (MOBI, AZW3, PDF)

Non-EPUB formats are handled by spawning [ExifTool](https://exiftool.org/) as a subprocess and reading the returned tag map.

Tag mapping:

| ExifTool tag | `BookMetadata` field | Fallback |
|--------------|---------------------|----------|
| `Title` | `Title` | Filename stem |
| `Author` | `Author` | `""` (empty string) |
| `ISBN` | `ISBN` | `Identifier` tag, then `""` (empty string) |

When ExifTool is **not installed**, `NewExtractor()` still returns a valid `*Extractor` (with a warning logged), but calling `ExtractMetadata` on a non-EPUB file returns an error:

```
exif-based metadata extraction requested but exiftool is not available
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

`cmd/cli` is a standalone wrapper around the extractor, useful for debugging metadata outside of the server.

```bash
# Build
go build -o biblioteka-cli ./cmd/cli

# Usage
./biblioteka-cli /path/to/book.epub
./biblioteka-cli /path/to/book.mobi
./biblioteka-cli /path/to/book.pdf
```

**Example output (EPUB with complete metadata):**

```json
{
  "Author": "Jane Austen",
  "Description": "A novel of manners set in rural England...",
  "Format": "EPUB",
  "ISBN": "9780141439518",
  "IsNative": true,
  "Language": "en",
  "PublicationDate": "1813-01-28",
  "Publisher": "Penguin Classics",
  "Title": "Pride and Prejudice"
}
```

**Example output (PDF without ExifTool):**

```
error: exif-based metadata extraction requested but exiftool is not available
```

**Example output (EPUB missing ISBN):**

```json
{
  "Author": "Some Author",
  "Description": "",
  "Format": "EPUB",
  "ISBN": "Not Found",
  "IsNative": true,
  "Publisher": "",
  "Title": "Some Book"
}
```

---

## What's next

The `process:file` background job ([`internal/jobs/process_file.go`](../internal/jobs/process_file.go)) now extracts and stores `Title`, `ISBN`, `Description`, `Publisher`, `Language`, `PublicationDate`, and `Author` (created/linked as a separate record) for EPUB files. Planned future improvements include:

1. **ExifTool field expansion** — extract `Description`, `Publisher`, `Language`, and `PublicationDate` from MOBI, AZW3, and PDF files via ExifTool so non-EPUB formats reach parity with the native EPUB parser.
2. **Page count** — extract and store page count for formats where that field is available.
3. **Cover image** — populate `cover_image_url` for formats that embed cover art.

Use `cmd/cli` to inspect what metadata Biblioteka would extract from a given file before it is imported.

---

## Contributing

To add support for a new field or format:

1. Edit [`internal/metadata/extractor.go`](../internal/metadata/extractor.go).
2. For EPUB: extend `extractNativeEpub` to read the relevant OPF element.
3. For ExifTool-based formats: add the appropriate `book.GetString("<TagName>")` call in `extractExif`.
4. Add or extend tests in `internal/metadata/extractor_test.go`.
5. Update the [Extracted fields](#extracted-fields) table in this document.
