-- migrate:up
ALTER TABLE passkey_challenges
ADD CONSTRAINT fk_passkey_challenges_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- migrate:down
ALTER TABLE passkey_challenges
DROP CONSTRAINT fk_passkey_challenges_user_id;
