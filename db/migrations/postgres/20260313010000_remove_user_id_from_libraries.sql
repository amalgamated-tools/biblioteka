-- migrate:up
DROP INDEX IF EXISTS idx_libraries_user_id;
ALTER TABLE libraries DROP CONSTRAINT IF EXISTS libraries_user_id_name_key;
ALTER TABLE libraries DROP COLUMN user_id;
ALTER TABLE libraries ADD CONSTRAINT libraries_name_key UNIQUE (name);

-- migrate:down
-- Irreversible: user_id ownership data cannot be restored after removal.
SELECT 1/0;
