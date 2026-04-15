-- migrate:up
ALTER TABLE ai_enrichments ADD CONSTRAINT chk_ai_enrichments_status CHECK (status IN ('pending', 'applied', 'rejected'));

-- migrate:down
ALTER TABLE ai_enrichments DROP CONSTRAINT chk_ai_enrichments_status;
