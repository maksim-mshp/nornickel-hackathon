CREATE UNIQUE INDEX numeric_facts_natural_key
    ON kg.numeric_facts (document_id, subject_id, parameter_id, condition_hash, operator, vmin, vmax, unit_code)
    NULLS NOT DISTINCT;
