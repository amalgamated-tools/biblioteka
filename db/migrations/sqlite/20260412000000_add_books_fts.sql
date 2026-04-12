-- migrate:up

-- Content table: FTS5 reads indexed content from the books table via rowid.
-- Using content= avoids duplicating text; using content_rowid=rowid ties FTS
-- entries to the corresponding books row.
--
-- NOTE: This index is tied to SQLite's implicit rowid, not the books.id TEXT
-- primary key. SQLite can reassign rowids during VACUUM on tables that lack an
-- INTEGER PRIMARY KEY alias. The books table uses a TEXT id so its rowid is
-- implicit. Running VACUUM (manually or via a maintenance routine) could
-- silently corrupt the FTS index by remapping rowids to wrong rows.
-- auto_vacuum is not enabled for this database, so this is safe in practice,
-- but any future maintenance that calls VACUUM should rebuild books_fts
-- afterwards (INSERT INTO books_fts(books_fts) VALUES ('rebuild')).
CREATE VIRTUAL TABLE books_fts USING fts5(
    title,
    description,
    content=books,
    content_rowid=rowid
);

-- Backfill existing rows into the FTS index.
INSERT INTO books_fts(rowid, title, description)
SELECT rowid, title, COALESCE(description, '') FROM books;

-- Keep the FTS index in sync with the books table.
CREATE TRIGGER books_fts_ai AFTER INSERT ON books BEGIN
    INSERT INTO books_fts(rowid, title, description) VALUES (new.rowid, new.title, COALESCE(new.description, ''));
END;

CREATE TRIGGER books_fts_ad AFTER DELETE ON books BEGIN
    INSERT INTO books_fts(books_fts, rowid, title, description)
        VALUES ('delete', old.rowid, old.title, COALESCE(old.description, ''));
END;

-- Only re-index when title or description actually changes; other column
-- updates (cover, ISBN, page count, etc.) do not affect the FTS index.
CREATE TRIGGER books_fts_au AFTER UPDATE ON books
WHEN old.title != new.title OR old.description IS NOT new.description BEGIN
    INSERT INTO books_fts(books_fts, rowid, title, description)
        VALUES ('delete', old.rowid, old.title, COALESCE(old.description, ''));
    INSERT INTO books_fts(rowid, title, description) VALUES (new.rowid, new.title, COALESCE(new.description, ''));
END;

-- migrate:down
DROP TRIGGER IF EXISTS books_fts_au;
DROP TRIGGER IF EXISTS books_fts_ad;
DROP TRIGGER IF EXISTS books_fts_ai;
DROP TABLE IF EXISTS books_fts;
