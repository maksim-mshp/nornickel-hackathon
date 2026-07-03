ALTER TABLE ops.answer_cache ADD COLUMN entity_slugs text[] NOT NULL DEFAULT '{}';
CREATE INDEX answer_cache_slugs_idx ON ops.answer_cache USING gin (entity_slugs);
