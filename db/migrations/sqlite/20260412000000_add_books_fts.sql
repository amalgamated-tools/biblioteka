-- migrate:up

-- Content table: FTS5 reads indexed content from the books table via rowid.
-- Using content= avoids duplicating text; using content_rowid=rowid ties FTS
-- entries to the corresponding books row.
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

CREATE TRIGGER books_fts_au AFTER UPDATE ON books BEGIN
    INSERT INTO books_fts(books_fts, rowid, title, description)
        VALUES ('delete', old.rowid, old.title, COALESCE(old.description, ''));
    INSERT INTO books_fts(rowid, title, description) VALUES (new.rowid, new.title, COALESCE(new.description, ''));
END;

-- migrate:down
DROP TRIGGER IF EXISTS books_fts_au;
DROP TRIGGER IF EXISTS books_fts_ad;
DROP TRIGGER IF EXISTS books_fts_ai;
DROP TABLE IF EXISTS books_fts;
