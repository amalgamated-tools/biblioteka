# Metadata Extraction

Biblioteka can extract metadata — title, author, ISBN, format — from book files using two complementary code paths:

| Path | Formats | External dependency |
|------|---------|---------------------|
| **Native EPUB** | `.epub` | None |
| **ExifTool** | `.mobi`, `.azw3`, `.pdf` | [ExifTool](https://exiftool.org/) must be on `PATH` |

The extractor is implemented in [`internal/metadata/extractor.go`](../internal/metadata/extractor.go) and exposed to end users via the standalone [`cmd/cli`](../cmd/cli/main.go) utility.

> **Import pipeline status:** Automatic metadata extraction during library scans (`process:file` background job) is **not yet implemented**. The job currently derives the book title from the filename only. Full integration is planned; see [Roadmap](#roadmap) below.

---

## Extracted fields

| Field | EPUB (native) | MOBI / AZW3 / PDF (ExifTool) | Notes |
|-------|:---:|:---:|-------|
| `Title` | ✓ | ✓ | Falls back to filename (without extension) when the ExifTool path cannot find a `Title` tag |
| `Author` | ✓ | ✓ | Falls back to `"Unknown"` when the ExifTool path cannot find an `Author` tag |
| `ISBN` | ✓ | ✓ | Returns `"Not Found"` when no valid 10- or 13-digit identifier is present; MOBI files also try an `Identifier` tag as a fallback |
| `Format` | ✓ | ✓ | Uppercase file extension (e.g. `"EPUB"`, `"PDF"`) |
| `IsNative` | `true` | `false` | Indicates whether the native EPUB parser was used |
| `Publisher` | ✗ | ✗ | Not yet extracted; always `""` |
| `Description` | ✗ | ✗ | Not yet extracted; always `""` |

---

## EPUB extraction (native)

EPUB files are parsed directly using the [`goreader/epub`](https://github.com/taylorskalyo/goreader) library — no external process is required.

The extractor reads the first OPF rootfile inside the ZIP container and maps:

| OPF field | `BookMetadata` field |
|-----------|---------------------|
| `<dc:title>` | `Title` |
| `<dc:creator>` | `Author` |
| `<dc:identifier>` | `ISBN` (cleaned of `urn:isbn:` / `isbn:` prefixes; validated as 10 or 13 digits) |

---

## ExifTool extraction (MOBI, AZW3, PDF)

Non-EPUB formats are handled by spawning [ExifTool](https://exiftool.org/) as a subprocess and reading the returned tag map.

Tag mapping:

| ExifTool tag | `BookMetadata` field | Fallback |
|--------------|---------------------|----------|
| `Title` | `Title` | Filename stem |
| `Author` | `Author` | `"Unknown"` |
| `ISBN` | `ISBN` | `Identifier` tag, then `"Not Found"` |

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
  "Description": "",
  "Format": "EPUB",
  "ISBN": "9780141439518",
  "IsNative": true,
  "Publisher": "",
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

## Roadmap

The `internal/metadata` package is designed to be called from the `process:file` background job ([`internal/jobs/process_file.go`](../internal/jobs/process_file.go)), which currently derives the book title from the filename only. The planned integration will:

1. Call `extractor.ExtractMetadata(path)` inside the `process:file` handler.
2. Populate the book record with the extracted `Title`, `Author`, `ISBN`, and other available fields.
3. Create an author record and link it to the book if an author name is found.

For now, use `cmd/cli` to manually inspect what metadata Biblioteka would extract from a given file.

---

## Contributing

To add support for a new field or format:

1. Edit [`internal/metadata/extractor.go`](../internal/metadata/extractor.go).
2. For EPUB: extend `extractNativeEpub` to read the relevant OPF element.
3. For ExifTool-based formats: add the appropriate `book.GetString("<TagName>")` call in `extractExif`.
4. Add or extend tests in `internal/metadata/extractor_test.go`.
5. Update the [Extracted fields](#extracted-fields) table in this document.
