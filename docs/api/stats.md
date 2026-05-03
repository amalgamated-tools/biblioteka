<!-- disable-agentic-editing: true -->

# API Reference — Stats & Recommendations

[← Back to API Reference](../api-reference.md)

## Stats

Statistical endpoints return per-user activity data. All stats endpoints require authentication.

### `GET /api/stats/downloads-per-month` 🔒

Returns monthly book-file download counts for the authenticated user over a rolling window.

**Query parameters:**

| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| `months` | integer | `12` | `24` | Number of calendar months to include, counting backwards from the current month (inclusive) |

Non-integer or sub-1 values for `months` default to `12`; values greater than `24` are clamped to `24`.

**Response:** `200 OK`

```json
[
  { "month": "2025-05", "count": 4 },
  { "month": "2025-06", "count": 7 },
  { "month": "2025-07", "count": 0 },
  ...
  { "month": "2026-04", "count": 12 }
]
```

The array always contains exactly `months` entries ordered oldest-first. Months with no downloads have `count: 0` — the series is never sparse. This makes the response safe to render directly as a bar chart without client-side gap-filling.

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `month` | string | Calendar month in `YYYY-MM` format |
| `count` | integer | Number of download events initiated by the authenticated user in that month |

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success |
| `401 Unauthorized` | Missing or invalid authentication |
| `405 Method Not Allowed` | Non-`GET` request |
| `500 Internal Server Error` | Database error |

> **User isolation:** Download counts are scoped to the authenticated user. Each user sees only their own download history.

---

### `GET /api/stats/year-in-books` 🔒

Returns annual reading and download statistics for the authenticated user for the specified calendar year. Useful for building "Year in Books" summary cards and streak visualizations.

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `year` | integer | Current year (UTC) | Calendar year to summarize. Must be between `1` and the current year + 1 (inclusive). Invalid values return `400 Bad Request`. |

**Response:** `200 OK`

```json
{
  "year": 2026,
  "books_finished": 14,
  "active_days": 87,
  "longest_streak": 12,
  "total_downloads": 31
}
```

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `year` | integer | The requested calendar year |
| `books_finished` | integer | Documents where reading `percentage >= 0.99` and last updated within the requested year (UTC) |
| `active_days` | integer | Number of distinct calendar dates (UTC) present in `reading_progress.updated_at` within the requested year |
| `longest_streak` | integer | Longest run of consecutive calendar dates (UTC) present in `reading_progress.updated_at` within the requested year |
| `total_downloads` | integer | Book file downloads initiated by the user within the year |

All counts return `0` when there is no activity for the requested year — the response is never `null` or absent.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success |
| `400 Bad Request` | `year` is not a positive integer or exceeds the current year + 1 |
| `401 Unauthorized` | Missing or invalid authentication |
| `405 Method Not Allowed` | Non-`GET` request |
| `500 Internal Server Error` | Database error |

> **User isolation:** Year-in-books statistics are scoped to the authenticated user. Each user sees only their own reading history.

---

### `GET /api/reading-progress/stats` 🔒

Returns the authenticated user's reading streak, total-books-tracked count, finished-books count, and a list of documents currently in progress. Progress data is sourced from KOReader sync (KOSync) read-position records.

**Response:** `200 OK`

```json
{
  "current_streak": 5,
  "total_tracked": 38,
  "total_finished": 21,
  "in_progress": [
    {
      "document": "my-book.epub",
      "percentage": 0.42,
      "device": "KOReader",
      "last_synced": "2026-04-13T18:00:00Z",
      "estimated_minutes_remaining": 74
    }
  ]
}
```

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `current_streak` | integer | Number of consecutive calendar days with at least one sync (streak resets if a day is missed) |
| `total_tracked` | integer | Total number of distinct documents with any recorded progress |
| `total_finished` | integer | Documents where `percentage >= 0.99` |
| `in_progress` | array | Documents with `0 < percentage < 0.99` |

**`in_progress` item fields:**

| Field | Type | Description |
|-------|------|-------------|
| `document` | string | Document filename as synced by KOReader |
| `percentage` | number | Reading progress in `[0, 1]` |
| `device` | string | KOReader device name, when provided; omitted otherwise |
| `last_synced` | string | ISO 8601 timestamp of the most recent sync |
| `estimated_minutes_remaining` | integer | Linear estimate of minutes left; omitted when the estimate is unreliable (≤ 1% read, < 5 min elapsed, or sync span exceeds 30 days, i.e. `updated_at − created_at > 30 days`) |

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success |
| `401 Unauthorized` | Missing or invalid authentication |
| `405 Method Not Allowed` | Non-`GET` request |
| `500 Internal Server Error` | Database error |

> **User isolation:** Progress stats are scoped to the authenticated user.

---

## Recommendations

The recommendations endpoint returns a ranked list of books the authenticated user has not yet read. Ranking is computed locally without any external service, using the user's Kobo reading history (books with `status` of `reading` or `finished`), and the internal score is not included in the JSON response.

### `GET /api/recommendations` 🔒

Returns a ranked list of books the authenticated user has not yet read as `bookSummaryDTO` objects. Results are ordered by an internal score derived from four signals based on the user's Kobo reading history; the per-book score is not exposed in the response body. When the user has no history all unread books are returned ordered newest-first.

**Ranking signals** (cumulative score per book):

| Signal | Weight | Description |
|--------|--------|-------------|
| Author overlap | +3 per shared author | Books sharing an author with a book the user is reading or has finished |
| Series continuation | +5 per series | The immediate next book in a series the user is reading or has finished |
| Publisher match | +1 | Books from a publisher the user has read before |
| Download popularity | `SUM(download_count) / 100` | Global download tiebreaker across the instance |

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | `10` | Maximum number of recommendations to return. Invalid, zero, or negative values use the default (`10`); values greater than `50` are capped at `50`. |
| `offset` | integer | `0` | Number of recommendations to skip before returning results. Useful for paginating through the full ranked list. Invalid or negative values default to `0`. |

**Response:** `200 OK`

An array of book summary objects. Returns an empty array (`[]`) when there are no unread books, never `null`.

```json
[
  {
    "id": "a1b2c3d4e5f6...",
    "title": "The Long Way to a Small, Angry Planet",
    "description": "A novel about ...",
    "asin": null,
    "isbn10": null,
    "isbn13": "9781473619814",
    "goodreads_id": "22733729",
    "hardcover_id": null,
    "google_books_id": null,
    "publication_date": "2014-08-13",
    "publisher": "Hodder & Stoughton",
    "language": "en",
    "cover_image_url": "https://example.com/covers/a1b2c3d4e5f6.jpg",
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-01-15T10:00:00Z"
  }
]
```

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Opaque resource ID |
| `title` | string | Book title |
| `description` | string \| null | Catalog description; `null` when absent |
| `asin` | string \| null | Amazon ASIN; `null` when absent |
| `isbn10` | string \| null | 10-digit ISBN; `null` when absent |
| `isbn13` | string \| null | 13-digit ISBN; `null` when absent |
| `goodreads_id` | string \| null | Goodreads book ID; `null` when absent |
| `hardcover_id` | string \| null | Hardcover.app book ID; `null` when absent |
| `google_books_id` | string \| null | Google Books volume ID; `null` when absent |
| `publication_date` | string \| null | Publication date in `YYYY-MM-DD` format; `null` when absent |
| `publisher` | string \| null | Publisher name; `null` when absent |
| `language` | string \| null | BCP 47 language code (e.g. `"en"`); `null` when absent |
| `cover_image_url` | string \| null | Stored cover image value, typically an HTTPS URL or `data:` URL; `null` when no cover is available |
| `created_at` | string | ISO 8601 timestamp when the book was added to the library |
| `updated_at` | string | ISO 8601 timestamp of the most recent metadata update |

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success (empty array when no recommendations are available) |
| `401 Unauthorized` | Missing or invalid authentication |
| `405 Method Not Allowed` | Non-`GET` request |
| `500 Internal Server Error` | Database error |

> **User isolation:** Recommendations are derived entirely from the authenticated user's own Kobo reading history. Users never see each other's reading data.

---

