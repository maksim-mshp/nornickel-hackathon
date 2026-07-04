BEGIN;

INSERT INTO kg.units (code, names, dimension, si_unit, si_factor, si_offset) VALUES
  ('m_per_s', ARRAY['м/с','m/s'], 'velocity', 'm/s', 1.0, 0),
  ('cm_per_s', ARRAY['см/с','cm/s'], 'velocity', 'm/s', 0.01, 0),
  ('mm_per_s', ARRAY['мм/с','mm/s'], 'velocity', 'm/s', 0.001, 0),
  ('m_per_min', ARRAY['м/мин','m/min'], 'velocity', 'm/s', 0.016666666666666666, 0),
  ('m_per_h', ARRAY['м/ч','m/h'], 'velocity', 'm/s', 0.0002777777777777778, 0),
  ('celsius', ARRAY['°C','℃','град','degC'], 'temperature', 'K', 1.0, 273.15),
  ('kelvin', ARRAY['K','К'], 'temperature', 'K', 1.0, 0),
  ('fahrenheit', ARRAY['°F','degF'], 'temperature', 'K', 0.5555555555555556, 255.3722222222222),
  ('pa', ARRAY['Па','Pa'], 'pressure', 'Pa', 1.0, 0),
  ('kpa', ARRAY['кПа','kPa'], 'pressure', 'Pa', 1000.0, 0),
  ('mpa', ARRAY['МПа','MPa'], 'pressure', 'Pa', 1000000.0, 0),
  ('bar', ARRAY['бар','bar'], 'pressure', 'Pa', 100000.0, 0),
  ('mbar', ARRAY['мбар','mbar'], 'pressure', 'Pa', 100.0, 0),
  ('atm', ARRAY['атм','atm'], 'pressure', 'Pa', 101325.0, 0),
  ('mmhg', ARRAY['мм рт. ст.','мм рт.ст.','ммрт.ст.','mmHg'], 'pressure', 'Pa', 133.322, 0),
  ('kgf_per_cm2', ARRAY['кгс/см²','кгс/см2'], 'pressure', 'Pa', 98066.5, 0),
  ('psi', ARRAY['psi','фунт/дюйм²'], 'pressure', 'Pa', 6894.757, 0),
  ('a_per_m2', ARRAY['А/м²','А/м2','A/m2'], 'current_density', 'A/m^2', 1.0, 0),
  ('a_per_dm2', ARRAY['А/дм²','А/дм2','A/dm2'], 'current_density', 'A/m^2', 100.0, 0),
  ('a_per_cm2', ARRAY['А/см²','А/см2','A/cm2'], 'current_density', 'A/m^2', 10000.0, 0),
  ('ma_per_cm2', ARRAY['мА/см²','мА/см2','mA/cm2'], 'current_density', 'A/m^2', 10.0, 0),
  ('ma_per_dm2', ARRAY['мА/дм²','мА/дм2','mA/dm2'], 'current_density', 'A/m^2', 0.1, 0),
  ('percent', ARRAY['%','проц.','об.%','об. %','мас.%','мас. %','масс.%','wt.%','wt%','ат.%','at.%','%об','%мас'], 'ratio', '%', 1.0, 0),
  ('permille', ARRAY['‰'], 'ratio', '%', 0.1, 0),
  ('ph', ARRAY['pH','рН','ph'], 'acidity', 'pH', 1.0, 0),
  ('mg_per_l', ARRAY['мг/л','mg/L','mg/l'], 'mass_concentration', 'kg/m^3', 0.001, 0),
  ('mg_per_dm3', ARRAY['мг/дм³','мг/дм3','mg/dm3'], 'mass_concentration', 'kg/m^3', 0.001, 0),
  ('g_per_l', ARRAY['г/л','g/L','g/l'], 'mass_concentration', 'kg/m^3', 1.0, 0),
  ('g_per_dm3', ARRAY['г/дм³','г/дм3','g/dm3'], 'mass_concentration', 'kg/m^3', 1.0, 0),
  ('kg_per_m3', ARRAY['кг/м³','кг/м3','kg/m3'], 'mass_concentration', 'kg/m^3', 1.0, 0),
  ('g_per_m3', ARRAY['г/м³','г/м3','g/m3'], 'mass_concentration', 'kg/m^3', 0.001, 0),
  ('mg_per_m3', ARRAY['мг/м³','мг/м3','mg/m3'], 'mass_concentration', 'kg/m^3', 1e-06, 0),
  ('mcg_per_l', ARRAY['мкг/л','µg/L','ug/L'], 'mass_concentration', 'kg/m^3', 1e-06, 0),
  ('ppm', ARRAY['ppm','млн⁻¹','мг/кг'], 'mass_fraction', 'kg/kg', 1e-06, 0),
  ('ppb', ARRAY['ppb','мкг/кг'], 'mass_fraction', 'kg/kg', 1e-09, 0),
  ('g_per_t', ARRAY['г/т','g/t'], 'mass_fraction', 'kg/kg', 1e-06, 0),
  ('mg_per_g', ARRAY['мг/г','mg/g'], 'mass_fraction', 'kg/kg', 0.001, 0),
  ('g_per_kg', ARRAY['г/кг','g/kg'], 'mass_fraction', 'kg/kg', 0.001, 0),
  ('mol_per_l', ARRAY['моль/л','моль/дм³','моль/дм3','mol/L'], 'molar_concentration', 'mol/m^3', 1000.0, 0),
  ('mmol_per_l', ARRAY['ммоль/л','mmol/L'], 'molar_concentration', 'mol/m^3', 1.0, 0),
  ('mol_per_m3', ARRAY['моль/м³','моль/м3','mol/m3'], 'molar_concentration', 'mol/m^3', 1.0, 0),
  ('t_per_day', ARRAY['т/сут','т/сутки','t/day'], 'mass_flow', 'kg/s', 0.011574074074074073, 0),
  ('t_per_h', ARRAY['т/ч','t/h'], 'mass_flow', 'kg/s', 0.2777777777777778, 0),
  ('t_per_year', ARRAY['т/год','t/year'], 'mass_flow', 'kg/s', 3.170979198376458e-05, 0),
  ('kg_per_h', ARRAY['кг/ч','kg/h'], 'mass_flow', 'kg/s', 0.0002777777777777778, 0),
  ('kg_per_s', ARRAY['кг/с','kg/s'], 'mass_flow', 'kg/s', 1.0, 0),
  ('m3_per_h', ARRAY['м³/ч','м3/ч','m3/h'], 'volumetric_flow', 'm^3/s', 0.0002777777777777778, 0),
  ('m3_per_day', ARRAY['м³/сут','м3/сут','m3/day'], 'volumetric_flow', 'm^3/s', 1.1574074074074073e-05, 0),
  ('m3_per_s', ARRAY['м³/с','м3/с','m3/s'], 'volumetric_flow', 'm^3/s', 1.0, 0),
  ('l_per_h', ARRAY['л/ч','L/h'], 'volumetric_flow', 'm^3/s', 2.777777777777778e-07, 0),
  ('l_per_min', ARRAY['л/мин','L/min'], 'volumetric_flow', 'm^3/s', 1.6666666666666667e-05, 0),
  ('l_per_s', ARRAY['л/с','L/s'], 'volumetric_flow', 'm^3/s', 0.001, 0),
  ('kwh_per_t', ARRAY['кВт·ч/т','кВтч/т','kWh/t'], 'specific_energy', 'J/kg', 3600.0, 0),
  ('kwh_per_kg', ARRAY['кВт·ч/кг','кВтч/кг','kWh/kg'], 'specific_energy', 'J/kg', 3600000.0, 0),
  ('mj_per_t', ARRAY['МДж/т','MJ/t'], 'specific_energy', 'J/kg', 1000.0, 0),
  ('kj_per_kg', ARRAY['кДж/кг','kJ/kg'], 'specific_energy', 'J/kg', 1000.0, 0),
  ('mm', ARRAY['мм','mm'], 'length', 'm', 0.001, 0),
  ('cm', ARRAY['см','cm'], 'length', 'm', 0.01, 0),
  ('um', ARRAY['мкм','µm','um'], 'length', 'm', 1e-06, 0),
  ('nm', ARRAY['нм','nm'], 'length', 'm', 1e-09, 0),
  ('hour', ARRAY['ч','час','часов','h'], 'duration', 's', 3600.0, 0),
  ('minute', ARRAY['мин','min'], 'duration', 's', 60.0, 0),
  ('second', ARRAY['сек','s'], 'duration', 's', 1.0, 0),
  ('day', ARRAY['сут','сутки','суток'], 'duration', 's', 86400.0, 0),
  ('rpm', ARRAY['об/мин','rpm'], 'rotational_speed', '1/s', 0.016666666666666666, 0),
  ('rps', ARRAY['об/с','rps'], 'rotational_speed', '1/s', 1.0, 0),
  ('rub_per_t', ARRAY['руб/т','руб./т','₽/т'], 'cost', 'currency', 1.0, 0),
  ('usd_per_t', ARRAY['$/т','USD/t'], 'cost', 'currency', 1.0, 0),
  ('eur_per_t', ARRAY['€/т','EUR/t'], 'cost', 'currency', 1.0, 0),
  ('rub_per_m3', ARRAY['руб/м³','руб/м3','₽/м³'], 'cost', 'currency', 1.0, 0)
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

INSERT INTO kg.entities (etype, canonical_name, canonical_name_en, slug, attrs) VALUES
  ('parameter', 'скорость потока', 'flow rate', 'parameter:flow-rate', '{}'),
  ('parameter', 'давление', 'pressure', 'parameter:pressure', '{}'),
  ('parameter', 'плотность тока', 'current density', 'parameter:current-density', '{}'),
  ('parameter', 'доля', 'ratio', 'parameter:ratio', '{}'),
  ('parameter', 'показатель pH', 'pH', 'parameter:ph', '{}'),
  ('parameter', 'концентрация', 'concentration', 'parameter:concentration', '{}'),
  ('parameter', 'содержание', 'content', 'parameter:content', '{}'),
  ('parameter', 'молярная концентрация', 'molar concentration', 'parameter:molar-concentration', '{}'),
  ('parameter', 'производительность', 'throughput', 'parameter:throughput', '{}'),
  ('parameter', 'объёмный расход', 'volumetric flow', 'parameter:volumetric-flow', '{}'),
  ('parameter', 'удельный расход энергии', 'energy intensity', 'parameter:energy-intensity', '{}'),
  ('parameter', 'размер', 'size', 'parameter:size', '{}'),
  ('parameter', 'длительность', 'duration', 'parameter:duration', '{}'),
  ('parameter', 'частота вращения', 'rotation speed', 'parameter:rotation-speed', '{}'),
  ('parameter', 'удельная стоимость', 'cost', 'parameter:cost', '{}')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO kg.parameter_defs (parameter_id, dimension, si_unit, plausible_min, plausible_max, notes)
SELECT id, dimension, si_unit, pmin, pmax, notes FROM (VALUES
  ('parameter:catholyte-flow-rate', 'velocity', 'm/s', 0, 20, 'circulation velocity'),
  ('parameter:temperature', 'temperature', 'K', 173, 2300, 'process temperature'),
  ('parameter:cathode-purity-gain', 'ratio', '%', 0, 100, 'relative gain'),
  ('parameter:sulfate-removal', 'ratio', '%', 0, 100, 'removal efficiency'),
  ('parameter:recovery', 'ratio', '%', 0, 100, 'permeate recovery'),
  ('parameter:flow-rate', 'velocity', 'm/s', 0, 50, 'flow velocity'),
  ('parameter:pressure', 'pressure', 'Pa', 0, 1000000000, 'process pressure'),
  ('parameter:current-density', 'current_density', 'A/m^2', 0, 100000, 'current density'),
  ('parameter:ratio', 'ratio', '%', 0, 1000, 'share or fraction'),
  ('parameter:ph', 'acidity', 'pH', 0, 14, 'acidity'),
  ('parameter:concentration', 'mass_concentration', 'kg/m^3', 0, 5000, 'mass concentration'),
  ('parameter:content', 'mass_fraction', 'kg/kg', 0, 1, 'mass fraction'),
  ('parameter:molar-concentration', 'molar_concentration', 'mol/m^3', 0, 100000, 'molar concentration'),
  ('parameter:throughput', 'mass_flow', 'kg/s', 0, 10000, 'mass throughput'),
  ('parameter:volumetric-flow', 'volumetric_flow', 'm^3/s', 0, 1000, 'volumetric flow'),
  ('parameter:energy-intensity', 'specific_energy', 'J/kg', 0, 1000000000, 'specific energy'),
  ('parameter:size', 'length', 'm', 0, 10000, 'linear size'),
  ('parameter:duration', 'duration', 's', 0, 1000000000, 'duration'),
  ('parameter:rotation-speed', 'rotational_speed', '1/s', 0, 100000, 'rotational speed')
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
