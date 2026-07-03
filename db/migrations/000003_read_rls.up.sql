ALTER TABLE kg.numeric_facts ENABLE ROW LEVEL SECURITY;
ALTER TABLE kg.numeric_facts FORCE ROW LEVEL SECURITY;

CREATE POLICY numeric_facts_select ON kg.numeric_facts FOR SELECT USING (
    EXISTS (SELECT 1 FROM core.documents d WHERE d.id = kg.numeric_facts.document_id)
);
CREATE POLICY numeric_facts_insert ON kg.numeric_facts FOR INSERT WITH CHECK (true);
CREATE POLICY numeric_facts_update ON kg.numeric_facts FOR UPDATE USING (true) WITH CHECK (true);

ALTER TABLE core.chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.chunks FORCE ROW LEVEL SECURITY;

CREATE POLICY chunks_select ON core.chunks FOR SELECT USING (
    EXISTS (SELECT 1 FROM core.documents d WHERE d.id = core.chunks.document_id)
);
CREATE POLICY chunks_insert ON core.chunks FOR INSERT WITH CHECK (true);
CREATE POLICY chunks_update ON core.chunks FOR UPDATE USING (true) WITH CHECK (true);
