BEGIN;

INSERT INTO kg.units (code, names, dimension, si_unit, si_factor, si_offset) VALUES
  ('m_per_s', ARRAY['м/с','m/s'], 'velocity', 'm/s', 1, 0),
  ('celsius', ARRAY['°C','C','град'], 'temperature', 'K', 1, 273.15),
  ('percent', ARRAY['%','мас.%','об.%'], 'ratio', '%', 1, 0),
  ('mg_per_l', ARRAY['мг/л','mg/L'], 'mass_concentration', 'kg/m^3', 0.001, 0),
  ('mg_per_dm3', ARRAY['мг/дм³','мг/дм3','mg/dm3'], 'mass_concentration', 'kg/m^3', 0.001, 0),
  ('a_per_m2', ARRAY['А/м²','A/m2'], 'current_density', 'A/m^2', 1, 0),
  ('mpa', ARRAY['МПа','MPa'], 'pressure', 'Pa', 1000000, 0),
  ('kelvin', ARRAY['K','К'], 'temperature', 'K', 1, 0)
ON CONFLICT (code) DO NOTHING;

INSERT INTO kg.entities (etype, canonical_name, canonical_name_en, slug, attrs) VALUES
  ('material', 'католит', 'catholyte', 'material:catholyte', '{"class":"solution"}'),
  ('material', 'сульфаты', 'sulfates', 'material:sulfates', '{"class":"salt"}'),
  ('material', 'хлориды', 'chlorides', 'material:chlorides', '{"class":"salt"}'),
  ('material', 'никелевая руда', 'nickel ore', 'material:nickel-ore', '{"class":"ore"}'),
  ('process', 'электроэкстракция никеля', 'nickel electrowinning', 'process:nickel-electrowinning', '{"domain":"hydro"}'),
  ('process', 'обессоливание воды', 'water desalination', 'process:desalination', '{"domain":"ecology"}'),
  ('process', 'кучное выщелачивание', 'heap leaching', 'process:heap-leaching', '{"domain":"hydro"}'),
  ('technology', 'обратный осмос', 'reverse osmosis', 'technology:reverse-osmosis', '{"trl":9}'),
  ('technology', 'ионный обмен', 'ion exchange', 'technology:ion-exchange', '{"trl":9}'),
  ('parameter', 'скорость циркуляции католита', 'catholyte flow rate', 'parameter:catholyte-flow-rate', '{}'),
  ('parameter', 'температура электролита', 'electrolyte temperature', 'parameter:temperature', '{}'),
  ('parameter', 'прирост чистоты катода', 'cathode purity gain', 'parameter:cathode-purity-gain', '{}'),
  ('parameter', 'степень удаления сульфатов', 'sulfate removal', 'parameter:sulfate-removal', '{}'),
  ('parameter', 'коэффициент выхода пермеата', 'permeate recovery', 'parameter:recovery', '{}'),
  ('property', 'сухой остаток', 'total dissolved solids', 'property:tds', '{"direction_good":"down"}'),
  ('experiment', 'эксперимент EXP-014', 'experiment EXP-014', 'experiment:exp-014', '{"series":"ЭН"}'),
  ('person', 'Иванов И. И.', 'Ivanov I. I.', 'person:ivanov', '{"position":"в.н.с."}'),
  ('person', 'Петрова А. А.', 'Petrova A. A.', 'person:petrova', '{"position":"н.с."}'),
  ('person', 'Смирнова Е. В.', 'Smirnova E. V.', 'person:smirnova', '{"position":"с.н.с."}'),
  ('lab', 'Лаборатория гидрометаллургии', 'Hydrometallurgy Lab', 'lab:hydrometallurgy', '{}'),
  ('lab', 'Лаборатория электрохимии', 'Electrochemistry Lab', 'lab:electrochemistry', '{}'),
  ('lab', 'Лаборатория водоподготовки', 'Water Treatment Lab', 'lab:water-treatment', '{}'),
  ('geography', 'Россия', 'Russia', 'geography:russia', '{"scope":"ru"}'),
  ('geography', 'зарубежная практика', 'foreign practice', 'geography:foreign', '{"scope":"foreign"}'),
  ('climate', 'холодный климат', 'cold climate', 'climate:cold', '{}')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO kg.parameter_defs (parameter_id, dimension, si_unit, plausible_min, plausible_max, notes)
SELECT id, dimension, si_unit, pmin, pmax, notes FROM (VALUES
  ('parameter:catholyte-flow-rate', 'velocity', 'm/s', 0, 20, 'circulation velocity'),
  ('parameter:temperature', 'temperature', 'K', 173, 2300, 'process temperature'),
  ('parameter:cathode-purity-gain', 'ratio', '%', 0, 100, 'relative gain'),
  ('parameter:sulfate-removal', 'ratio', '%', 0, 100, 'removal efficiency'),
  ('parameter:recovery', 'ratio', '%', 0, 100, 'permeate recovery')
) AS defs(slug, dimension, si_unit, pmin, pmax, notes)
JOIN kg.entities e ON e.slug = defs.slug
ON CONFLICT (parameter_id) DO NOTHING;

INSERT INTO kg.entity_aliases (entity_id, alias, lang, source)
SELECT id, alias, lang, 'dictionary' FROM (VALUES
  ('process:nickel-electrowinning', 'электровыделение никеля', 'ru'),
  ('process:nickel-electrowinning', 'electrowinning', 'en'),
  ('property:tds', 'солесодержание', 'ru'),
  ('property:tds', 'TDS', 'en'),
  ('technology:reverse-osmosis', 'RO', 'en'),
  ('material:catholyte', 'circulating catholyte', 'en')
) AS a(slug, alias, lang)
JOIN kg.entities e ON e.slug = a.slug
ON CONFLICT (entity_id, alias, lang) DO NOTHING;

INSERT INTO core.documents (id, title, doc_type, lang, year, geography, access_level, sha256, status) VALUES
  ('a1000000-0000-4000-8000-000000000017', 'Отчёт: оптимизация циркуляции католита', 'report', 'ru', 2023, 'foreign', 'internal', decode('d017','hex'), 'indexed'),
  ('a1000000-0000-4000-8000-000000000042', 'Протокол опытной серии ЭН-7', 'protocol', 'ru', 2021, 'ru', 'internal', decode('d042','hex'), 'indexed'),
  ('a1000000-0000-4000-8000-000000000058', 'Отчёт: режимы диафрагменных ячеек', 'report', 'ru', 2023, 'ru', 'internal', decode('d058','hex'), 'indexed'),
  ('a1000000-0000-4000-8000-000000000101', 'Nickel electrowinning practice review', 'article', 'en', 2022, 'foreign', 'internal', decode('d101','hex'), 'indexed'),
  ('a1000000-0000-4000-8000-000000000201', 'Обзор технологий обессоливания оборотных вод', 'article', 'en', 2022, 'foreign', 'internal', decode('d201','hex'), 'indexed'),
  ('a1000000-0000-4000-8000-000000000215', 'Ионообменная очистка сточных вод обогатительной фабрики', 'report', 'ru', 2023, 'ru', 'internal', decode('d215','hex'), 'indexed'),
  ('a1000000-0000-4000-8000-000000000310', 'Heap leaching of nickel laterites', 'article', 'en', 2019, 'foreign', 'internal', decode('d310','hex'), 'indexed')
ON CONFLICT (id) DO NOTHING;

INSERT INTO core.document_versions (document_id, version, blob_uri, parser_version, parsed_at)
SELECT id, 1, 's3://kmap-raw/' || id || '/1', 'seed-1.0', now() FROM core.documents
WHERE id::text LIKE 'a1000000-%'
ON CONFLICT (document_id, version) DO NOTHING;

INSERT INTO kg.numeric_facts
  (id, document_id, subject_id, parameter_id, operator, value_raw, vmin, vmax, unit_orig, unit_code,
   vmin_si, vmax_si, conditions, quote, page, geography, doc_year, extraction_method, extractor_version,
   extraction_confidence, validation_status)
SELECT f.id::uuid, f.document_id::uuid, s.id, p.id, f.operator::kg.op, f.value_raw, f.vmin, f.vmax, f.unit_orig, f.unit_code,
   f.vmin_si, f.vmax_si, f.conditions::jsonb, f.quote, f.page, f.geography::core.geo_scope, f.doc_year,
   f.method::kg.extraction_method, f.extractor_version, f.confidence, f.validation::kg.validation_status
FROM (VALUES
  ('f0000000-0000-4000-8000-000000000001','a1000000-0000-4000-8000-000000000017','process:nickel-electrowinning','parameter:catholyte-flow-rate','range','0,8–1,0 м/с',0.8,1.0,'м/с','m_per_s',0.8,1.0,'{"температура":"60–70 °C","плотность тока":"220 А/м²"}','при скорости циркуляции католита 0,8–1,0 м/с достигалась максимальная равномерность осаждения и чистота катода',12,'foreign',2023,'deterministic','numcore-1.4.0',0.98,'multi_source'),
  ('f0000000-0000-4000-8000-000000000002','a1000000-0000-4000-8000-000000000042','process:nickel-electrowinning','parameter:catholyte-flow-rate','gt','выше 0,7 м/с',0.7,NULL,'м/с','m_per_s',0.7,NULL,'{"температура":"55–60 °C","плотность тока":"320 А/м²"}','при скорости потока выше 0,7 м/с наблюдался рост дефектности катодного осадка',4,'ru',2021,'deterministic','numcore-1.4.0',0.97,'contradicted'),
  ('f0000000-0000-4000-8000-000000000003','a1000000-0000-4000-8000-000000000058','process:nickel-electrowinning','parameter:catholyte-flow-rate','eq','0,8 м/с',0.8,0.8,'м/с','m_per_s',0.8,0.8,'{"температура":"65 °C","среда":"сульфатная"}','скорость циркуляции католита составляла 0,8 м/с',21,'ru',2023,'deterministic','numcore-1.4.0',0.99,'expert_validated'),
  ('f0000000-0000-4000-8000-000000000004','a1000000-0000-4000-8000-000000000101','process:nickel-electrowinning','parameter:catholyte-flow-rate','range','0.6–0.9 m/s',0.6,0.9,'м/с','m_per_s',0.6,0.9,'{"температура":"60–80 °C"}','catholyte circulation velocities of 0.6–0.9 m/s are typical in modern tankhouses',7,'foreign',2022,'deterministic','numcore-1.4.0',0.98,'multi_source'),
  ('f0000000-0000-4000-8000-000000000005','a1000000-0000-4000-8000-000000000017','process:nickel-electrowinning','parameter:temperature','range','60–70 °C',60,70,'°C','celsius',333.15,343.15,'{}','температура электролита поддерживалась в диапазоне 60–70 °C',9,'foreign',2023,'deterministic','numcore-1.4.0',0.98,'multi_source'),
  ('f0000000-0000-4000-8000-000000000006','a1000000-0000-4000-8000-000000000058','experiment:exp-014','parameter:cathode-purity-gain','eq','4,2 %',4.2,4.2,'%','percent',4.2,4.2,'{"скорость потока":"0.8 м/с","температура":"65 °C"}','чистота катода повысилась на 4,2 % относительно базового режима',23,'ru',2023,'catalog','catalog-1.0',0.99,'expert_validated'),
  ('f0000000-0000-4000-8000-000000000011','a1000000-0000-4000-8000-000000000201','technology:reverse-osmosis','property:tds','lte','не более 500 мг/дм³',NULL,500,'мг/дм³','mg_per_dm3',NULL,0.5,'{"исходное солесодержание":"1000–1500 мг/дм³"}','обратный осмос обеспечивает сухой остаток не более 500 мг/дм³ на пермеате',15,'foreign',2022,'deterministic','numcore-1.4.0',0.98,'multi_source'),
  ('f0000000-0000-4000-8000-000000000012','a1000000-0000-4000-8000-000000000215','technology:ion-exchange','parameter:sulfate-removal','gte','не менее 95 %',95,NULL,'%','percent',95,NULL,'{"сульфаты на входе":"200–300 мг/л"}','степень удаления сульфат-ионов ионным обменом составляла не менее 95 %',8,'ru',2023,'deterministic','numcore-1.4.0',0.97,'expert_validated'),
  ('f0000000-0000-4000-8000-000000000013','a1000000-0000-4000-8000-000000000201','technology:reverse-osmosis','parameter:recovery','range','70–80 %',70,80,'%','percent',70,80,'{"давление":"1.5–2.0 МПа"}','рабочий коэффициент выхода пермеата составляет 70–80 %',17,'foreign',2022,'deterministic','numcore-1.4.0',0.96,'multi_source'),
  ('f0000000-0000-4000-8000-000000000021','a1000000-0000-4000-8000-000000000310','process:heap-leaching','parameter:temperature','gte','at or above 15 °C',15,NULL,'°C','celsius',288.15,NULL,'{"климат":"умеренный"}','effective bacterial leaching requires temperatures at or above 15 °C',4,'foreign',2019,'deterministic','numcore-1.4.0',0.95,'weak_evidence')
) AS f(id, document_id, subject_slug, parameter_slug, operator, value_raw, vmin, vmax, unit_orig, unit_code,
       vmin_si, vmax_si, conditions, quote, page, geography, doc_year, method, extractor_version, confidence, validation)
JOIN kg.entities s ON s.slug = f.subject_slug
JOIN kg.entities p ON p.slug = f.parameter_slug
ON CONFLICT (id) DO NOTHING;

INSERT INTO kg.edges (src, dst, rel, weight, confidence)
SELECT a.id, b.id, e.rel, e.weight, e.confidence FROM (VALUES
  ('process:nickel-electrowinning','material:catholyte','USES_MATERIAL',4,0.9),
  ('process:nickel-electrowinning','parameter:catholyte-flow-rate','OPERATES_AT',4,0.95),
  ('process:nickel-electrowinning','parameter:temperature','OPERATES_AT',1,0.9),
  ('experiment:exp-014','parameter:cathode-purity-gain','PRODUCES_PROPERTY',1,0.99),
  ('experiment:exp-014','process:nickel-electrowinning','USES_PROCESS',1,0.95),
  ('process:desalination','technology:reverse-osmosis','USES_PROCESS',1,0.9),
  ('process:desalination','technology:ion-exchange','USES_PROCESS',1,0.9),
  ('technology:reverse-osmosis','property:tds','IMPROVES',2,0.9),
  ('process:heap-leaching','material:nickel-ore','USES_MATERIAL',1,0.8)
) AS e(src_slug, dst_slug, rel, weight, confidence)
JOIN kg.entities a ON a.slug = e.src_slug
JOIN kg.entities b ON b.slug = e.dst_slug
ON CONFLICT (src, dst, rel) DO NOTHING;

INSERT INTO kg.edges (src, dst, rel, weight, confidence)
SELECT a.id, b.id, 'CONTRADICTS', 1, 0.86 FROM kg.entities a, kg.entities b
WHERE a.slug = 'process:nickel-electrowinning' AND b.slug = 'parameter:catholyte-flow-rate'
ON CONFLICT (src, dst, rel) DO NOTHING;

INSERT INTO kg.edges (src, dst, rel, weight, confidence)
SELECT a.id, b.id, e.rel, 1, 0.9 FROM (VALUES
  ('person:ivanov','lab:hydrometallurgy','AFFILIATED'),
  ('person:petrova','lab:electrochemistry','AFFILIATED'),
  ('person:smirnova','lab:water-treatment','AFFILIATED'),
  ('person:ivanov','process:nickel-electrowinning','WORKED_ON'),
  ('person:petrova','process:nickel-electrowinning','WORKED_ON'),
  ('person:smirnova','process:desalination','WORKED_ON')
) AS e(src_slug, dst_slug, rel)
JOIN kg.entities a ON a.slug = e.src_slug
JOIN kg.entities b ON b.slug = e.dst_slug
ON CONFLICT (src, dst, rel) DO NOTHING;

INSERT INTO epi.clusters (id, ckey, subject_id, parameter_id, kind, condition_class, size, dirty)
SELECT c.id::uuid, decode(c.ckey, 'hex'), s.id, p.id, 'numeric', c.condition_class::jsonb, c.size, false FROM (VALUES
  ('c0000000-0000-4000-8000-000000000001','cc01','process:nickel-electrowinning','parameter:catholyte-flow-rate','{"temperature_c":"60-90","current_density_a_m2":"150-250"}',4),
  ('c0000000-0000-4000-8000-000000000002','cc02','technology:reverse-osmosis','property:tds','{"medium":"sulfate"}',2)
) AS c(id, ckey, subject_slug, parameter_slug, condition_class, size)
JOIN kg.entities s ON s.slug = c.subject_slug
JOIN kg.entities p ON p.slug = c.parameter_slug
ON CONFLICT (id) DO NOTHING;

INSERT INTO epi.cluster_members (cluster_id, fact_kind, fact_id) VALUES
  ('c0000000-0000-4000-8000-000000000001','numeric','f0000000-0000-4000-8000-000000000001'),
  ('c0000000-0000-4000-8000-000000000001','numeric','f0000000-0000-4000-8000-000000000002'),
  ('c0000000-0000-4000-8000-000000000001','numeric','f0000000-0000-4000-8000-000000000003'),
  ('c0000000-0000-4000-8000-000000000001','numeric','f0000000-0000-4000-8000-000000000004'),
  ('c0000000-0000-4000-8000-000000000002','numeric','f0000000-0000-4000-8000-000000000011')
ON CONFLICT (cluster_id, fact_kind, fact_id) DO NOTHING;

INSERT INTO epi.consensus (cluster_id, verdict, agreed_range, overlap_index, stats, confidence, engine_version) VALUES
  ('c0000000-0000-4000-8000-000000000001','majority',numrange(0.8,0.9,'[]'),0.42,'{"unit":"м/с","parameter":"скорость циркуляции католита","sources":[{"title":"Отчёт 2023 (doc_017)","year":2023,"geography":"foreign","vmin":0.8,"vmax":1.0},{"title":"Протокол ЭН-7 (doc_042)","year":2021,"geography":"ru","vmin":0.3,"vmax":0.7},{"title":"Отчёт 2023 (doc_058)","year":2023,"geography":"ru","vmin":0.8,"vmax":0.8},{"title":"Review 2022 (doc_101)","year":2022,"geography":"foreign","vmin":0.6,"vmax":0.9}]}',0.78,'epi-1.0'),
  ('c0000000-0000-4000-8000-000000000002','consensus',numrange(300,500,'[]'),0.61,'{"unit":"мг/дм³","parameter":"сухой остаток","sources":[{"title":"Обзор 2022 (doc_201)","year":2022,"geography":"foreign","vmin":300,"vmax":500},{"title":"Отчёт 2023 (doc_215)","year":2023,"geography":"ru","vmin":350,"vmax":500}]}',0.85,'epi-1.0')
ON CONFLICT (cluster_id) DO NOTHING;

INSERT INTO epi.contradictions (id, cluster_id, a_kind, a_id, b_kind, b_id, dtype, status, severity, judge_model, judge_rationale, confounders) VALUES
  ('e0000000-0000-4000-8000-000000000001','c0000000-0000-4000-8000-000000000001','numeric','f0000000-0000-4000-8000-000000000001','numeric','f0000000-0000-4000-8000-000000000002','range_disjoint','judge_confirmed',0.7,'openai-gpt-oss-120b','режимы отличаются плотностью тока; при 320 А/м² высокая скорость даёт дефекты','["плотность тока","температура электролита"]')
ON CONFLICT (a_kind, a_id, b_kind, b_id) DO NOTHING;

INSERT INTO epi.coverage_cells (domain, material_id, process_id, condition_key, docs, experiments, facts, experts, ru_docs, foreign_docs, validated_facts, last_source_year, score, score_components, gap_flag, gap_reasons)
SELECT d.domain, m.id, p.id, d.condition_key, d.docs, d.experiments, d.facts, d.experts, d.ru_docs, d.foreign_docs, d.validated_facts, d.last_year, d.score, d.components::jsonb, d.gap_flag, d.gap_reasons::text[] FROM (VALUES
  ('hydro','material:catholyte','process:nickel-electrowinning','',4,2,5,2,2,2,3,2023,78.0,'{"docs":0.8,"experiments":0.6,"experts":0.7,"recency":0.9,"validated":0.7}',false,ARRAY[]::text[]),
  ('hydro','material:catholyte','process:nickel-electrowinning','current_density:high',1,0,1,1,0,1,0,2022,18.0,'{"docs":0.3,"experiments":0.0,"experts":0.4,"recency":0.7,"validated":0.0}',true,ARRAY['no_experiments','foreign_only']),
  ('ecology','material:sulfates','process:desalination','',2,0,3,1,1,1,2,2023,52.0,'{"docs":0.5,"experiments":0.2,"experts":0.5,"recency":0.9,"validated":0.6}',false,ARRAY[]::text[]),
  ('ecology','material:sulfates','process:desalination','magnesium:high',0,0,0,0,0,0,0,NULL,22.0,'{"docs":0.2,"experiments":0.0,"experts":0.2,"recency":0.0,"validated":0.0}',true,ARRAY['no_ru_practice','low_validation']),
  ('hydro','material:nickel-ore','process:heap-leaching','climate:cold',1,0,1,0,0,1,0,2019,6.0,'{"docs":0.1,"experiments":0.0,"experts":0.0,"recency":0.4,"validated":0.0}',true,ARRAY['no_experiments','no_ru_practice','stale'])
) AS d(domain, material_slug, process_slug, condition_key, docs, experiments, facts, experts, ru_docs, foreign_docs, validated_facts, last_year, score, components, gap_flag, gap_reasons)
JOIN kg.entities m ON m.slug = d.material_slug
JOIN kg.entities p ON p.slug = d.process_slug
ON CONFLICT (domain, material_id, process_id, condition_key) DO NOTHING;

INSERT INTO epi.expert_topics (person_id, entity_id, weight, evidence)
SELECT pe.id, en.id, t.weight, t.evidence::jsonb FROM (VALUES
  ('person:ivanov','process:nickel-electrowinning',0.83,'{"lab":"Лаборатория гидрометаллургии","reports":7,"experiments":12,"last_year":2025,"documents":["a1000000-0000-4000-8000-000000000017"]}'),
  ('person:petrova','process:nickel-electrowinning',0.76,'{"lab":"Лаборатория электрохимии","reports":5,"experiments":8,"last_year":2024,"documents":["a1000000-0000-4000-8000-000000000058"]}'),
  ('person:smirnova','process:desalination',0.79,'{"lab":"Лаборатория водоподготовки","reports":6,"experiments":9,"last_year":2024,"documents":["a1000000-0000-4000-8000-000000000215"]}')
) AS t(person_slug, entity_slug, weight, evidence)
JOIN kg.entities pe ON pe.slug = t.person_slug
JOIN kg.entities en ON en.slug = t.entity_slug
ON CONFLICT (person_id, entity_id) DO NOTHING;

INSERT INTO iam.users (oidc_sub, display_name, email, roles, doc_access) VALUES
  ('demo-researcher', 'Исследователь', 'researcher@kmap.local', ARRAY['researcher'], 'internal'),
  ('demo-manager', 'Руководитель', 'manager@kmap.local', ARRAY['manager'], 'confidential'),
  ('demo-expert', 'Эксперт', 'expert@kmap.local', ARRAY['expert'], 'confidential'),
  ('demo-admin', 'Администратор', 'admin@kmap.local', ARRAY['admin'], 'restricted'),
  ('demo-partner', 'Внешний партнёр', 'partner@kmap.local', ARRAY['external_partner'], 'public')
ON CONFLICT (oidc_sub) DO NOTHING;

COMMIT;
