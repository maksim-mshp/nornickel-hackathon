DROP INDEX IF EXISTS ops.answer_cache_slugs_idx;
ALTER TABLE ops.answer_cache DROP COLUMN IF EXISTS entity_slugs;
