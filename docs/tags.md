# Tags

Tags are user-defined labels you apply to books for categorization and discovery. A tag is a globally-scoped named entity — not per-user — so any label you create is available to everyone on the instance. A book can carry many tags, and a single tag can span many books.

Common uses:

- Genre and sub-genre classification ("science fiction", "hard sf", "space opera")
- Reading-status labels ("to-read", "did-not-finish")
- Theme or mood labels ("cozy", "dark", "hopeful")
- Custom collections ("book-club", "gift ideas", "award winners")

---

## Browsing the Tags page

Open **Tags** from the main sidebar to see every tag on the instance, sorted alphabetically. Each tag shows its name; click the pencil icon to rename it or the trash icon to delete it.

---

## Creating a tag

1. Click **New Tag** on the Tags page.
2. Type a name (e.g. `mystery`) and press **Enter** or click **Create**.

The server normalizes the name before saving it: leading and trailing whitespace is trimmed, and any internal run of whitespace is collapsed to a single space. Capitalization is preserved. For example, `"  hard   sf  "` is stored as `"hard sf"`.

Creating a tag that already exists (case-insensitive comparison) returns a `409 Conflict` error.

---

## Renaming a tag

1. Click the pencil icon next to the tag on the Tags page.
2. Edit the name in the text field.
3. Press **Enter** or click the checkmark to save, or press **Escape** to cancel.

The same name normalization rules apply. Renaming a tag updates it everywhere — all books that carry the tag immediately reflect the new name.

---

## Deleting a tag

1. Click the trash icon next to the tag on the Tags page.
2. Confirm the deletion in the dialog that appears.

Deleting a tag removes it from every book that carries it. Books themselves are not deleted.

---

## Assigning tags to a book

1. Open a book's detail page (click the book title anywhere in your library).
2. Locate the **Tags** field in the book metadata panel.
3. Click inside the tags field to open the tag picker.
4. Type to filter existing tags, then click the tag you want to add.
5. To add a tag that does not exist yet, type its name and click **Create "[name]"** at the bottom of the dropdown list.
6. To remove a tag from the book, click the × next to its name.

Changes are saved automatically as you add or remove tags.

---

## AI-assisted tagging

When AI metadata enrichment is enabled (requires an Ollama instance configured by your admin), the **Enrich with AI** action on a book's detail page asks the model to suggest genres, themes, and tags. Suggested tags are union-merged with the book's existing tags — new tags are created automatically via `FindOrCreate` if they do not yet exist, and no existing tags are removed.

---

## API reference

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/tags` | List all tags (sorted by name) |
| `POST` | `/api/tags` | Create a tag |
| `GET` | `/api/tags/{id}` | Get a single tag |
| `PUT` | `/api/tags/{id}` | Rename a tag |
| `DELETE` | `/api/tags/{id}` | Delete a tag |
| `GET` | `/api/books/{id}/tags` | List tags assigned to a book |
| `PUT` | `/api/books/{id}/tags` | Replace all tags on a book |

All endpoints require authentication (JWT, API key, or session cookie). See the [API Reference](api/tags.md) for full request and response schemas.

### Quick example: create a tag

```http
POST /api/tags
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "name": "science fiction"
}
```

**Response `201 Created`:**

```json
{
  "id": "01abc...",
  "name": "science fiction",
  "created_at": "2026-04-18T19:00:00Z",
  "updated_at": "2026-04-18T19:00:00Z"
}
```

### Quick example: assign tags to a book

```http
PUT /api/books/{id}/tags
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "tag_ids": ["01abc...", "01def..."]
}
```

Returns `200 OK` with the updated list of tag objects.
