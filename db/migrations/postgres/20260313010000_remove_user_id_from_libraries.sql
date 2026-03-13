-- migrate:up
DROP INDEX IF EXISTS idx_libraries_user_id;
ALTER TABLE libraries DROP CONSTRAINT IF EXISTS libraries_user_id_name_key;
ALTER TABLE libraries DROP COLUMN user_id;
ALTER TABLE libraries ADD CONSTRAINT libraries_name_key UNIQUE (name);

-- migrate:down
ALTER TABLE libraries DROP CONSTRAINT IF EXISTS libraries_name_key;
ALTER TABLE libraries ADD COLUMN user_id TEXT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE libraries ADD CONSTRAINT libraries_user_id_name_key UNIQUE (user_id, name);
CREATE INDEX idx_libraries_user_id ON libraries(user_id);
