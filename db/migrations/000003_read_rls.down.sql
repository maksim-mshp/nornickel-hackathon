DROP POLICY IF EXISTS chunks_update ON core.chunks;
DROP POLICY IF EXISTS chunks_insert ON core.chunks;
DROP POLICY IF EXISTS chunks_select ON core.chunks;
ALTER TABLE core.chunks NO FORCE ROW LEVEL SECURITY;
ALTER TABLE core.chunks DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS numeric_facts_update ON kg.numeric_facts;
DROP POLICY IF EXISTS numeric_facts_insert ON kg.numeric_facts;
DROP POLICY IF EXISTS numeric_facts_select ON kg.numeric_facts;
ALTER TABLE kg.numeric_facts NO FORCE ROW LEVEL SECURITY;
ALTER TABLE kg.numeric_facts DISABLE ROW LEVEL SECURITY;
