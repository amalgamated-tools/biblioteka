-- migrate:up
CREATE TABLE reading_list_books (
    reading_list_id TEXT     NOT NULL REFERENCES reading_lists(id) ON DELETE CASCADE,
    book_id         TEXT     NOT NULL REFERENCES books(id)         ON DELETE CASCADE,
    position        INTEGER  NOT NULL DEFAULT 0,
    added_at        DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (reading_list_id, book_id)
);

CREATE INDEX idx_reading_list_books_book ON reading_list_books(book_id);

-- migrate:down
DROP TABLE reading_list_books;
