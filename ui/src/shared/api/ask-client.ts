import type {
  AnswerDoc,
  AskEvent,
  Consensus,
  Contradiction,
  EntityRef,
  EvidencePack,
  EvidenceStats,
  Expert,
  Fact,
  NumericValue,
  Provenance,
  QueryPlan,
} from "@/shared/api/types";
import {
  CATHOLYTE_ANSWER,
  CATHOLYTE_PACK,
  CATHOLYTE_PLAN,
  CATHOLYTE_SUMMARY,
} from "@/shared/api/mock/catholyte-scenario";
import { authHeaders } from "@/shared/lib/role";

const ASK_ENDPOINT = "/v1/ask";
const MAX_CONNECT_RETRIES = 2;
const RETRY_BASE_MS = 400;
const IDLE_TIMEOUT_MS = 30_000;

async function connectAsk(question: string, signal?: AbortSignal): Promise<Response | null> {
  for (let attempt = 0; ; attempt++) {
    try {
      const response = await fetch(ASK_ENDPOINT, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "text/event-stream",
          ...authHeaders(),
        },
        body: JSON.stringify({ question }),
        signal,
      });
      if (response.ok && response.body) return response;
      if (attempt >= MAX_CONNECT_RETRIES) return null;
    } catch (error) {
      if ((error as Error).name === "AbortError") throw error;
      if (attempt >= MAX_CONNECT_RETRIES) return null;
    }
    await sleep(RETRY_BASE_MS * 2 ** attempt, signal);
  }
}

export async function* askStream(
  question: string,
  signal?: AbortSignal,
): AsyncGenerator<AskEvent> {
  const response = await connectAsk(question, signal);
  if (!response?.body) {
    yield* mockAskStream(question, signal);
    return;
  }
  yield* parseSSE(response.body, signal);
}

function readWithTimeout(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  ms: number,
  signal?: AbortSignal,
): Promise<ReadableStreamReadResult<Uint8Array>> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(
      () => reject(new DOMException("idle-timeout", "TimeoutError")),
      ms,
    );
    const onAbort = () => {
      clearTimeout(timer);
      reject(new DOMException("aborted", "AbortError"));
    };
    if (signal?.aborted) {
      onAbort();
      return;
    }
    signal?.addEventListener("abort", onAbort, { once: true });
    reader.read().then(
      (result) => {
        clearTimeout(timer);
        signal?.removeEventListener("abort", onAbort);
        resolve(result);
      },
      (error) => {
        clearTimeout(timer);
        signal?.removeEventListener("abort", onAbort);
        reject(error);
      },
    );
  });
}

async function* parseSSE(
  body: ReadableStream<Uint8Array>,
  signal?: AbortSignal,
): AsyncGenerator<AskEvent> {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      let result: ReadableStreamReadResult<Uint8Array>;
      try {
        result = await readWithTimeout(reader, IDLE_TIMEOUT_MS, signal);
      } catch (error) {
        if ((error as Error).name === "AbortError") throw error;
        yield { type: "error", message: "Ответ прервался: превышено время ожидания" };
        return;
      }
      const { value, done } = result;
      if (done) break;
      buffer += decoder.decode(value, { stream: true });

      let boundary = buffer.indexOf("\n\n");
      while (boundary !== -1) {
        const frame = buffer.slice(0, boundary);
        buffer = buffer.slice(boundary + 2);
        const event = parseFrame(frame);
        if (event) yield event;
        boundary = buffer.indexOf("\n\n");
      }
    }
  } finally {
    await reader.cancel().catch(() => {});
  }
}

function asArray<T>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : [];
}

function obj(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" ? (value as Record<string, unknown>) : {};
}

function num(value: unknown, fallback = 0): number {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

function normalizeEntityRef(value: unknown): EntityRef {
  const o = obj(value);
  return { slug: String(o.slug ?? ""), name: String(o.name ?? "") };
}

function normalizeNumericValue(value: unknown): NumericValue {
  const o = obj(value);
  return {
    operator: (o.operator as NumericValue["operator"]) ?? "eq",
    vmin: typeof o.vmin === "number" ? o.vmin : undefined,
    vmax: typeof o.vmax === "number" ? o.vmax : undefined,
    unit: String(o.unit ?? ""),
  };
}

function normalizeProvenance(value: unknown): Provenance {
  const o = obj(value);
  return {
    documentId: String(o.documentId ?? ""),
    title: String(o.title ?? ""),
    docType: String(o.docType ?? ""),
    page: num(o.page),
    quote: String(o.quote ?? ""),
    year: num(o.year),
  };
}

function normalizeFact(value: unknown): Fact {
  const o = obj(value);
  const sc = obj(o.scoreComponents);
  return {
    id: String(o.id ?? ""),
    ref: String(o.ref ?? ""),
    subject: normalizeEntityRef(o.subject),
    parameter: normalizeEntityRef(o.parameter),
    value: normalizeNumericValue(o.value),
    si: normalizeNumericValue(o.si ?? o.value),
    conditions:
      o.conditions && typeof o.conditions === "object"
        ? (o.conditions as Record<string, string>)
        : {},
    geography: (o.geography as Fact["geography"]) ?? "unknown",
    provenance: normalizeProvenance(o.provenance),
    extractionMethod: (o.extractionMethod as Fact["extractionMethod"]) ?? "deterministic",
    extractorVersion: String(o.extractorVersion ?? ""),
    confidence: num(o.confidence),
    validationStatus: (o.validationStatus as Fact["validationStatus"]) ?? "machine_extracted",
    score: num(o.score),
    scoreComponents: {
      match: num(sc.match),
      rerank: num(sc.rerank),
      source: num(sc.source),
      validation: num(sc.validation),
      freshness: num(sc.freshness),
    },
  };
}

const PLAN_INTENTS: QueryPlan["intent"][] = [
  "technology_search",
  "experiment_search",
  "literature_review",
  "expert_search",
  "gap_analysis",
  "contradiction_analysis",
  "comparison",
  "entity_lookup",
];

function normalizePlan(value: unknown): QueryPlan {
  const o = obj(value);
  const e = obj(o.entities);
  const intent = PLAN_INTENTS.includes(o.intent as QueryPlan["intent"])
    ? (o.intent as QueryPlan["intent"])
    : "technology_search";
  return {
    intent,
    entities: {
      materials: asArray<unknown>(e.materials).map(normalizeEntityRef),
      processes: asArray<unknown>(e.processes).map(normalizeEntityRef),
      properties: asArray<unknown>(e.properties).map(normalizeEntityRef),
    },
    paramConstraints: asArray<unknown>(o.paramConstraints).map((c) => {
      const co = obj(c);
      return {
        parameter: normalizeEntityRef(co.parameter),
        value: normalizeNumericValue(co.value),
      };
    }),
    geography: (o.geography as QueryPlan["geography"]) ?? "any",
    yearFrom: typeof o.yearFrom === "number" ? o.yearFrom : undefined,
    yearTo: typeof o.yearTo === "number" ? o.yearTo : undefined,
    parser: o.parser === "rules" ? "rules" : "llm",
    confidence: num(o.confidence),
  };
}

export function normalizeAnswer(value: unknown): AnswerDoc {
  const o = obj(value);
  const g = obj(o.guard);
  return {
    summary: String(o.summary ?? ""),
    confidence: num(o.confidence),
    methods: asArray<unknown>(o.methods).map((m) => {
      const mo = obj(m);
      return {
        name: String(mo.name ?? ""),
        applicability: String(mo.applicability ?? ""),
        citations: asArray<string>(mo.citations),
      };
    }),
    guard: {
      numbersChecked: num(g.numbersChecked),
      violations: num(g.violations),
      degraded: Boolean(g.degraded ?? false),
    },
  };
}

export function normalizePack(data: Record<string, unknown>): EvidencePack {
  const gaps = asArray<Record<string, unknown>>(data.gaps).map((gap) => ({
    label: String(gap.label ?? ""),
    score: num(gap.score),
    reasons: asArray<string>(gap.reasons),
    neighbors: asArray<string>(gap.neighbors),
  }));
  const stats = (data.stats as EvidenceStats | undefined) ?? {
    sources: 0,
    ruSources: 0,
    foreignSources: 0,
    yearFrom: 0,
    yearTo: 0,
  };
  return {
    facts: asArray<unknown>(data.facts).map(normalizeFact),
    consensus: asArray<Consensus>(data.consensus),
    contradictions: asArray<Contradiction>(data.contradictions),
    gaps,
    experts: asArray<Expert>(data.experts),
    stats,
  };
}

function parseFrame(frame: string): AskEvent | null {
  let event = "";
  const dataLines: string[] = [];
  for (const line of frame.split("\n")) {
    if (line.startsWith("event:")) {
      event = line.slice(6).trim();
    } else if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trimStart());
    }
  }
  if (!event || dataLines.length === 0) return null;

  let data: unknown;
  try {
    data = JSON.parse(dataLines.join("\n"));
  } catch {
    return null;
  }
  const d = obj(data);
  switch (event) {
    case "plan":
      return { type: "plan", plan: normalizePlan(data) };
    case "evidence":
      return { type: "evidence", pack: normalizePack(d) };
    case "answer.delta":
      return { type: "answer.delta", text: String(d.text ?? "") };
    case "answer.done":
      return { type: "answer.done", answer: normalizeAnswer(data) };
    case "error":
      return {
        type: "error",
        message: String(d.detail ?? d.title ?? d.message ?? "Ошибка ответа"),
      };
    default:
      return null;
  }
}

const PLAN_DELAY_MS = 600;
const EVIDENCE_DELAY_MS = 900;
const DELTA_DELAY_MS = 90;
const DELTA_WORDS = 8;
const DONE_DELAY_MS = 400;

function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(resolve, ms);
    signal?.addEventListener("abort", () => {
      clearTimeout(timer);
      reject(new DOMException("aborted", "AbortError"));
    });
  });
}

function splitDeltas(text: string): string[] {
  const words = text.match(/\S+\s*/g) ?? [];
  const chunks: string[] = [];
  for (let i = 0; i < words.length; i += DELTA_WORDS) {
    chunks.push(words.slice(i, i + DELTA_WORDS).join(""));
  }
  return chunks;
}

export async function* mockAskStream(
  question: string,
  signal?: AbortSignal,
): AsyncGenerator<AskEvent> {
  void question;
  await sleep(PLAN_DELAY_MS, signal);
  yield { type: "plan", plan: CATHOLYTE_PLAN };
  await sleep(EVIDENCE_DELAY_MS, signal);
  yield { type: "evidence", pack: CATHOLYTE_PACK };
  for (const delta of splitDeltas(CATHOLYTE_SUMMARY)) {
    await sleep(DELTA_DELAY_MS, signal);
    yield { type: "answer.delta", text: delta };
  }
  await sleep(DONE_DELAY_MS, signal);
  yield { type: "answer.done", answer: CATHOLYTE_ANSWER };
}
