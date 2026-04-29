-- migrate:up
-- Postgres already has idx_books_created_at defined as (created_at DESC, id DESC)
-- from migration 20260411222140. Rename for naming consistency with SQLite.
ALTER INDEX idx_books_created_at RENAME TO idx_books_created_at_id;

-- migrate:down
ALTER INDEX idx_books_created_at_id RENAME TO idx_books_created_at;
