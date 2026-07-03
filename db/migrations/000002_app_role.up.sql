DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kmap_app') THEN
        CREATE ROLE kmap_app LOGIN PASSWORD 'kmap_app';
    END IF;
END
$$;

GRANT USAGE ON SCHEMA core, kg, epi, iam, ops, eval TO kmap_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA core, kg, epi, iam, ops, eval TO kmap_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA core, kg, epi, iam, ops, eval TO kmap_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA core, kg, epi, iam, ops, eval GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kmap_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA core, kg, epi, iam, ops, eval GRANT USAGE, SELECT ON SEQUENCES TO kmap_app;
