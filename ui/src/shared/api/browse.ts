import { authHeaders } from "@/shared/lib/role";

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
    const response = await fetch(path, {
      headers: { Accept: "application/json", ...authHeaders() },
    });
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

export async function getDocuments(): Promise<DocumentRow[]> {
  try {
    const all: DocumentRow[] = [];
    let cursor = "";
    for (let page = 0; page < 100; page++) {
      const url = cursor
        ? `/v1/documents?limit=100&cursor=${encodeURIComponent(cursor)}`
        : "/v1/documents?limit=100";
      const response = await fetch(url, {
        headers: { Accept: "application/json", ...authHeaders() },
      });
      if (!response.ok) return all;
      const data = (await response.json()) as {
        items?: DocumentRow[];
        next_cursor?: string;
      };
      if (Array.isArray(data.items)) all.push(...data.items);
      cursor = data.next_cursor ?? "";
      if (!cursor) break;
    }
    return all;
  } catch {
    return DOCUMENTS;
  }
}

export async function openDocumentSource(id: string): Promise<boolean> {
  try {
    const response = await fetch(`/v1/documents/${encodeURIComponent(id)}/file`, {
      headers: { ...authHeaders() },
    });
    if (!response.ok) return false;
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    window.open(url, "_blank", "noopener");
    setTimeout(() => URL.revokeObjectURL(url), 60_000);
    return true;
  } catch {
    return false;
  }
}

export type CoverageCellLive = {
  material: string;
  process: string;
  score: number;
  facts: number;
  experiments: number;
  reasons: string[];
};

type CoverageApiCell = {
  material?: string;
  process?: string;
  score?: number;
  gap_flag?: boolean;
  reasons?: string[] | null;
  counters?: { facts?: number; experiments?: number; material?: string; process?: string };
};

export async function getCoverageCells(): Promise<CoverageCellLive[] | null> {
  const items = await getJSON<CoverageApiCell[] | null>("/v1/coverage", null);
  if (!Array.isArray(items) || items.length === 0) return null;
  return items.map((cell) => ({
    material: cell.counters?.material ?? cell.material ?? "",
    process: cell.counters?.process ?? cell.process ?? "",
    score: Math.round(cell.score ?? 0),
    facts: cell.counters?.facts ?? 0,
    experiments: cell.counters?.experiments ?? 0,
    reasons: cell.reasons ?? [],
  }));
}

export type ContradictionLive = {
  id: string;
  status: string;
  subject: string;
  parameter: string;
  aStatement: string;
  bStatement: string;
  cause: string;
  severity: number;
};

export async function getContradictions(status = ""): Promise<ContradictionLive[]> {
  const path = status
    ? `/v1/contradictions?status=${encodeURIComponent(status)}`
    : "/v1/contradictions";
  const items = await getJSON<Partial<ContradictionLive>[]>(path, []);
  if (!Array.isArray(items)) return [];
  return items.map((item) => ({
    id: item.id ?? "",
    status: item.status ?? "",
    subject: item.subject ?? "",
    parameter: item.parameter ?? "",
    aStatement: item.aStatement ?? "",
    bStatement: item.bStatement ?? "",
    cause: item.cause ?? "",
    severity: item.severity ?? 0,
  }));
}

export type ReviewEntity = {
  id: string;
  slug: string;
  name: string;
  type: string;
  candidate: string;
  similarity: number;
};

export type ReviewOrphan = {
  id: string;
  value: string;
  quote: string;
  status: string;
  candidates: string[];
};

export async function getReviewEntities(): Promise<ReviewEntity[]> {
  const items = await getJSON<Partial<ReviewEntity>[]>(
    "/v1/review/queue?kind=entities",
    [],
  );
  if (!Array.isArray(items)) return [];
  return items.map((item) => ({
    id: item.id ?? "",
    slug: item.slug ?? "",
    name: item.name ?? "",
    type: item.type ?? "",
    candidate: item.candidate ?? "",
    similarity: item.similarity ?? 0,
  }));
}

export async function getReviewOrphans(): Promise<ReviewOrphan[]> {
  const items = await getJSON<Partial<ReviewOrphan>[]>(
    "/v1/review/queue?kind=orphans",
    [],
  );
  if (!Array.isArray(items)) return [];
  return items.map((item) => ({
    id: item.id ?? "",
    value: item.value ?? "",
    quote: item.quote ?? "",
    status: item.status ?? "",
    candidates: item.candidates ?? [],
  }));
}

export async function updateFactStatus(
  id: string,
  status: string,
  comment = "",
): Promise<boolean> {
  try {
    const response = await fetch(`/v1/facts/${encodeURIComponent(id)}/status`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({ status, comment }),
    });
    return response.ok;
  } catch {
    return false;
  }
}

export async function decideContradiction(
  id: string,
  decision: "confirmed" | "rejected",
  comment = "",
): Promise<boolean> {
  try {
    const response = await fetch(
      `/v1/contradictions/${encodeURIComponent(id)}/decision`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ decision, comment }),
      },
    );
    return response.ok;
  } catch {
    return false;
  }
}

export type EntitySummaryLive = {
  id: string;
  slug: string;
  name: string;
  nameEn: string;
  etype: string;
};

const ETYPE_LABELS: Record<string, string> = {
  material: "материал",
  process: "процесс",
  equipment: "оборудование",
  property: "свойство",
  parameter: "параметр",
  technology: "технология",
  experiment: "эксперимент",
  publication: "публикация",
  person: "эксперт",
  lab: "лаборатория",
  org: "организация",
  geography: "география",
  topic: "тема",
  economic_indicator: "экономика",
  climate: "климат",
  facility: "объект",
};

type EntityCardApi = {
  id?: string;
  slug?: string;
  nameRu?: string;
  nameEn?: string;
  type?: string;
  synonyms?: string[] | null;
  counters?: Record<string, number> | null;
  experts?: Partial<ExpertProfile>[] | null;
  timeline?: { year?: number; facts?: number }[] | null;
};

type GraphNodeApi = { id?: string; label?: string };
type GraphEdgeApi = { src?: string; dst?: string; rel?: string; weight?: number };
type GraphApi = { nodes?: GraphNodeApi[] | null; edges?: GraphEdgeApi[] | null };

const REL_LABELS: Record<string, string> = {
  OPERATES_AT: "параметры",
  USES_MATERIAL: "материалы",
  USES_EQUIPMENT: "оборудование",
  USES_PROCESS: "процессы",
  PRODUCES_PROPERTY: "свойства",
  IMPROVES: "влияние",
  APPLICABLE_FOR: "применимость",
  MENTIONED_IN: "упоминания",
  RELATED_TO: "связи",
};

async function graphRelations(
  anchorId: string,
  slug: string,
): Promise<{ group: string; items: { slug: string; name: string; weight: number }[] }[]> {
  const graph = await getJSON<GraphApi | null>(
    `/v1/graph?entity_id=${encodeURIComponent(slug)}&depth=1&top_n=40`,
    null,
  );
  if (!graph || !Array.isArray(graph.edges)) return [];
  const labels = new Map<string, string>();
  for (const node of graph.nodes ?? []) {
    if (node.id) labels.set(node.id, node.label ?? labelFromSlug(node.id));
  }
  const groups = new Map<string, { group: string; items: { slug: string; name: string; weight: number }[] }>();
  for (const edge of graph.edges) {
    const neighbor = edge.src === anchorId ? edge.dst : edge.src;
    if (!neighbor || neighbor === anchorId) continue;
    const rel = edge.rel ?? "RELATED_TO";
    const group = REL_LABELS[rel] ?? rel.toLowerCase();
    if (!groups.has(group)) groups.set(group, { group, items: [] });
    const items = groups.get(group)!.items;
    if (items.some((existing) => existing.slug === neighbor)) continue;
    items.push({ slug: neighbor, name: labels.get(neighbor) ?? labelFromSlug(neighbor), weight: edge.weight ?? 0.5 });
  }
  return [...groups.values()];
}

function labelFromSlug(slug: string): string {
  const after = slug.includes(":") ? slug.slice(slug.indexOf(":") + 1) : slug;
  return after.replace(/-/g, " ") || slug;
}

export type EntityCardLive = {
  slug: string;
  type: string;
  nameRu: string;
  nameEn: string;
  synonyms: { value: string; pending: boolean }[];
  counters: { documents: number; facts: number; experiments: number; experts: number };
  relations: { group: string; items: { slug: string; name: string; weight: number }[] }[];
  experts: ExpertProfile[];
  timeline: { year: number; facts: number }[];
};

export async function getEntityCard(slug: string): Promise<EntityCardLive | null> {
  const data = await getJSON<EntityCardApi | null>(
    `/v1/entities/${encodeURIComponent(slug)}`,
    null,
  );
  if (!data || (!data.slug && !data.nameRu && !data.id)) return null;

  const counters = data.counters ?? {};
  const relations = data.id ? await graphRelations(data.id, data.slug ?? slug) : [];

  return {
    slug: data.slug ?? slug,
    type: ETYPE_LABELS[data.type ?? ""] ?? data.type ?? "сущность",
    nameRu: data.nameRu ?? slug,
    nameEn: data.nameEn ?? "",
    synonyms: (data.synonyms ?? []).map((value) => ({ value, pending: false })),
    counters: {
      documents: counters.documents ?? 0,
      facts: counters.facts ?? 0,
      experiments: counters.experiments ?? 0,
      experts: counters.experts ?? 0,
    },
    relations,
    experts: (data.experts ?? []).map((item) => ({
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
    })),
    timeline: (data.timeline ?? []).map((point) => ({
      year: point.year ?? 0,
      facts: point.facts ?? 0,
    })),
  };
}

export async function getEntities(query = ""): Promise<EntitySummaryLive[]> {
  const path = query
    ? `/v1/entities?q=${encodeURIComponent(query)}`
    : "/v1/entities";
  const items = await getJSON<Partial<EntitySummaryLive>[]>(path, []);
  if (!Array.isArray(items)) return [];
  return items.map((item) => ({
    id: item.id ?? "",
    slug: item.slug ?? "",
    name: item.name ?? "",
    nameEn: item.nameEn ?? "",
    etype: item.etype ?? "",
  }));
}
