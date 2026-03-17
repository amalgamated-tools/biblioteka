-- migrate:up
CREATE TABLE IF NOT EXISTS kosync_credentials (
    id            TEXT        PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id       TEXT        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    username      TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_kosync_credentials_username ON kosync_credentials (LOWER(username));
CREATE INDEX IF NOT EXISTS idx_kosync_credentials_user_id ON kosync_credentials (user_id);

CREATE TABLE IF NOT EXISTS reading_progress (
    id         TEXT             PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id    TEXT             NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document   TEXT             NOT NULL,
    progress   TEXT             NOT NULL,
    percentage DOUBLE PRECISION NOT NULL DEFAULT 0,
    device     TEXT,
    device_id  TEXT,
    created_at TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_reading_progress_user_document ON reading_progress (user_id, document);
CREATE INDEX IF NOT EXISTS idx_reading_progress_user_id ON reading_progress (user_id);

-- migrate:down
DROP TABLE IF EXISTS reading_progress;
DROP TABLE IF EXISTS kosync_credentials;
