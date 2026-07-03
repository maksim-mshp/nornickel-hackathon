ALTER DEFAULT PRIVILEGES IN SCHEMA core, kg, epi, iam, ops, eval REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM kmap_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA core, kg, epi, iam, ops, eval REVOKE USAGE, SELECT ON SEQUENCES FROM kmap_app;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA core, kg, epi, iam, ops, eval FROM kmap_app;
REVOKE ALL ON ALL TABLES IN SCHEMA core, kg, epi, iam, ops, eval FROM kmap_app;
REVOKE USAGE ON SCHEMA core, kg, epi, iam, ops, eval FROM kmap_app;
DROP ROLE IF EXISTS kmap_app;
