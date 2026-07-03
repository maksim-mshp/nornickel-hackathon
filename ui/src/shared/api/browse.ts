export type ExpertProfile = {
  id: string;
  name: string;
  lab: string;
  weight: number;
  reports: number;
  experiments: number;
  lastYear: number;
  topics: string[];
  activity: { year: number; count: number }[];
  evidence: { title: string; year: number; kind: string }[];
};

export type ExperimentRow = {
  id: string;
  code: string;
  material: string;
  process: string;
  conditions: Record<string, string>;
  result: string;
  source: string;
  docType: string;
  confidence: number;
};

export type DocumentRow = {
  id: string;
  title: string;
  docType: string;
  lang: string;
  geography: string;
  accessLevel: string;
  status: string;
  facts: number;
  year: number;
};

async function getJSON<T>(path: string, fallback: T): Promise<T> {
  try {
    const response = await fetch(path, { headers: { Accept: "application/json" } });
    if (!response.ok) return fallback;
    const data = (await response.json()) as Partial<Record<string, unknown>>;
    const items = (data as { items?: T }).items;
    return (items ?? (data as T)) ?? fallback;
  } catch {
    return fallback;
  }
}

const EXPERTS: ExpertProfile[] = [
  {
    id: "person:ivanov",
    name: "Иванов И. И.",
    lab: "Лаборатория гидрометаллургии",
    weight: 0.83,
    reports: 7,
    experiments: 12,
    lastYear: 2025,
    topics: ["электроэкстракция никеля", "циркуляция католита"],
    activity: [
      { year: 2021, count: 2 },
      { year: 2022, count: 3 },
      { year: 2023, count: 4 },
      { year: 2024, count: 2 },
      { year: 2025, count: 1 },
    ],
    evidence: [
      { title: "Отчёт: оптимизация циркуляции католита", year: 2023, kind: "отчёт" },
      { title: "EXP-014 · режимы диафрагменных ячеек", year: 2023, kind: "эксперимент" },
    ],
  },
  {
    id: "person:petrova",
    name: "Петрова А. А.",
    lab: "Лаборатория электрохимии",
    weight: 0.76,
    reports: 5,
    experiments: 8,
    lastYear: 2024,
    topics: ["электроэкстракция никеля", "плотность тока"],
    activity: [
      { year: 2021, count: 1 },
      { year: 2022, count: 2 },
      { year: 2023, count: 3 },
      { year: 2024, count: 2 },
    ],
    evidence: [
      { title: "Отчёт: режимы диафрагменных ячеек", year: 2023, kind: "отчёт" },
    ],
  },
  {
    id: "person:smirnova",
    name: "Смирнова Е. В.",
    lab: "Лаборатория водоподготовки",
    weight: 0.79,
    reports: 6,
    experiments: 9,
    lastYear: 2024,
    topics: ["обессоливание воды", "ионный обмен"],
    activity: [
      { year: 2020, count: 1 },
      { year: 2022, count: 3 },
      { year: 2023, count: 3 },
      { year: 2024, count: 2 },
    ],
    evidence: [
      { title: "Ионообменная очистка сточных вод", year: 2023, kind: "отчёт" },
    ],
  },
];

const EXPERIMENTS: ExperimentRow[] = [
  {
    id: "experiment:exp-014",
    code: "EXP-014",
    material: "католит",
    process: "электроэкстракция никеля",
    conditions: { "скорость потока": "0,8 м/с", температура: "65 °C", среда: "сульфатная" },
    result: "чистота катода +4,2 %",
    source: "Отчёт: режимы диафрагменных ячеек",
    docType: "report",
    confidence: 0.99,
  },
  {
    id: "experiment:en-7",
    code: "ЭН-7",
    material: "католит",
    process: "электроэкстракция никеля",
    conditions: { "скорость потока": ">0,7 м/с", "плотность тока": "320 А/м²", температура: "55–60 °C" },
    result: "рост дефектности осадка",
    source: "Протокол опытной серии ЭН-7",
    docType: "protocol",
    confidence: 0.97,
  },
  {
    id: "experiment:ro-1",
    code: "RO-1",
    material: "сульфаты",
    process: "обессоливание воды",
    conditions: { давление: "1,5–2,0 МПа", "сухой остаток": "≤500 мг/дм³" },
    result: "выход пермеата 70–80 %",
    source: "Обзор технологий обессоливания",
    docType: "article",
    confidence: 0.96,
  },
  {
    id: "experiment:ie-3",
    code: "ИО-3",
    material: "сульфаты",
    process: "обессоливание воды",
    conditions: { "сульфаты на входе": "200–300 мг/л" },
    result: "удаление сульфатов ≥95 %",
    source: "Ионообменная очистка сточных вод",
    docType: "report",
    confidence: 0.97,
  },
];

const DOCUMENTS: DocumentRow[] = [
  { id: "doc_017", title: "Отчёт: оптимизация циркуляции католита", docType: "report", lang: "ru", geography: "foreign", accessLevel: "internal", status: "indexed", facts: 2, year: 2023 },
  { id: "doc_042", title: "Протокол опытной серии ЭН-7", docType: "protocol", lang: "ru", geography: "ru", accessLevel: "internal", status: "indexed", facts: 1, year: 2021 },
  { id: "doc_058", title: "Отчёт: режимы диафрагменных ячеек", docType: "report", lang: "ru", geography: "ru", accessLevel: "internal", status: "indexed", facts: 2, year: 2023 },
  { id: "doc_101", title: "Nickel electrowinning practice review", docType: "article", lang: "en", geography: "foreign", accessLevel: "internal", status: "indexed", facts: 1, year: 2022 },
  { id: "doc_201", title: "Обзор технологий обессоливания оборотных вод", docType: "article", lang: "en", geography: "foreign", accessLevel: "internal", status: "indexed", facts: 2, year: 2022 },
  { id: "doc_215", title: "Ионообменная очистка сточных вод обогатительной фабрики", docType: "report", lang: "ru", geography: "ru", accessLevel: "internal", status: "indexed", facts: 1, year: 2023 },
  { id: "doc_310", title: "Heap leaching of nickel laterites", docType: "article", lang: "en", geography: "foreign", accessLevel: "internal", status: "indexed", facts: 1, year: 2019 },
];

export async function getExperts(
  entityId = "process:nickel-electrowinning",
): Promise<ExpertProfile[]> {
  const items = await getJSON<Partial<ExpertProfile>[]>(
    `/v1/experts?entity_id=${encodeURIComponent(entityId)}`,
    EXPERTS,
  );
  if (!Array.isArray(items) || items.length === 0) return EXPERTS;
  return items.map((item) => ({
    id: item.id ?? "",
    name: item.name ?? "",
    lab: item.lab ?? "",
    weight: item.weight ?? 0,
    reports: item.reports ?? 0,
    experiments: item.experiments ?? 0,
    lastYear: item.lastYear ?? 0,
    topics: item.topics ?? [],
    activity: item.activity ?? [],
    evidence: item.evidence ?? [],
  }));
}

export function getExperiments(): Promise<ExperimentRow[]> {
  return getJSON("/v1/experiments", EXPERIMENTS);
}

export function getDocuments(): Promise<DocumentRow[]> {
  return getJSON("/v1/documents", DOCUMENTS);
}
