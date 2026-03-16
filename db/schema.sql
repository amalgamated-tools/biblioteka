CREATE TABLE schema_migrations (
version TEXT PRIMARY KEY,
applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE users (
id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
name TEXT NOT NULL,
email TEXT NOT NULL UNIQUE COLLATE NOCASE,
password_hash TEXT NOT NULL,
oidc_subject TEXT,
is_admin INTEGER NOT NULL DEFAULT 0,
created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX idx_users_oidc_subject ON users(oidc_subject) WHERE oidc_subject IS NOT NULL;
CREATE TABLE settings (
key TEXT PRIMARY KEY,
value TEXT NOT NULL,
updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE libraries (
id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
name TEXT NOT NULL UNIQUE,
paths TEXT NOT NULL DEFAULT '[]',
organization_type TEXT NOT NULL DEFAULT 'book_per_folder',
monitored INTEGER NOT NULL DEFAULT 0,
created_at DATETIME NOT NULL DEFAULT (datetime('now')),
updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE authors (
id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
name TEXT NOT NULL UNIQUE,
goodreads_id TEXT,
hardcover_id TEXT,
google_books_id TEXT,
image_url TEXT,
created_at DATETIME NOT NULL DEFAULT (datetime('now')),
updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX idx_authors_name_ci ON authors (LOWER(name));
CREATE TABLE series (
id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
name TEXT NOT NULL UNIQUE,
goodreads_id TEXT,
hardcover_id TEXT,
google_books_id TEXT,
created_at DATETIME NOT NULL DEFAULT (datetime('now')),
updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE books (
id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
title TEXT NOT NULL,
description TEXT,
asin TEXT,
isbn10 TEXT,
isbn13 TEXT,
goodreads_id TEXT,
hardcover_id TEXT,
google_books_id TEXT,
publication_date TEXT,
publisher TEXT,
language TEXT,
num_pages INTEGER,
cover_image_url TEXT,
created_at DATETIME NOT NULL DEFAULT (datetime('now')),
updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE library_books (
library_id TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
created_at DATETIME NOT NULL DEFAULT (datetime('now')),
PRIMARY KEY (library_id, book_id)
);
CREATE INDEX idx_library_books_book_id ON library_books(book_id);
CREATE TABLE book_authors (
book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
author_id TEXT NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
created_at DATETIME NOT NULL DEFAULT (datetime('now')),
PRIMARY KEY (book_id, author_id)
);
CREATE INDEX idx_book_authors_author_id ON book_authors(author_id);
CREATE TABLE book_series (
book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
series_id TEXT NOT NULL REFERENCES series(id) ON DELETE CASCADE,
position REAL,
created_at DATETIME NOT NULL DEFAULT (datetime('now')),
PRIMARY KEY (book_id, series_id)
);
CREATE INDEX idx_book_series_series_id ON book_series(series_id);
CREATE TABLE book_files (
id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
file_type TEXT NOT NULL,
file_name TEXT NOT NULL,
file_size INTEGER NOT NULL,
file_hash TEXT,
file_path TEXT NOT NULL,
created_at DATETIME NOT NULL DEFAULT (datetime('now')),
updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_book_files_book_id ON book_files(book_id);
CREATE TABLE audit_logs (
id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
user_id TEXT,
action TEXT NOT NULL,
entity_type TEXT NOT NULL,
entity_id TEXT NOT NULL,
metadata TEXT,
created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs (entity_type, entity_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);
CREATE TABLE api_keys (
id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
name TEXT NOT NULL,
key_hash TEXT NOT NULL,
key_prefix TEXT NOT NULL,
last_used_at DATETIME,
created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX idx_api_keys_key_hash ON api_keys (key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys (user_id);
CREATE TABLE opds_credentials (
id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
username TEXT NOT NULL UNIQUE COLLATE NOCASE,
password_hash TEXT NOT NULL,
created_at DATETIME NOT NULL DEFAULT (datetime('now')),
updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_opds_credentials_username ON opds_credentials (LOWER(username));
CREATE INDEX idx_opds_credentials_user_id ON opds_credentials (user_id);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20260214235631_create_users_table'),
  ('20260222200000_create_settings_table'),
  ('20260222210000_add_oidc_to_users'),
  ('20260224000000_add_is_admin_to_users'),
  ('20260313000000_create_libraries_table'),
  ('20260313010000_remove_user_id_from_libraries'),
  ('20260313020000_create_authors_table'),
  ('20260313030000_create_series_table'),
  ('20260313040000_create_books_table'),
  ('20260313050000_create_library_books_table'),
  ('20260313060000_create_book_authors_table'),
  ('20260313070000_create_book_series_table'),
  ('20260313080000_create_book_files_table'),
  ('20260314000000_create_audit_logs_table'),
  ('20260315000000_create_api_keys_table'),
  ('20260315000000_create_opds_credentials_table'),
  ('20260316000000_author_name_ci_unique');
