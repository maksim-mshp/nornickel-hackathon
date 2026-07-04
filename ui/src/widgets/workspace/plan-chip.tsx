import Link from "next/link";
import { FactValue } from "@/entities/fact/fact-value";
import type { QueryPlan } from "@/shared/api/types";

function slugPrefix(slug: string): string {
  const idx = slug.indexOf(":");
  return idx > 0 ? slug.slice(0, idx) : "сущность";
}

const INTENT_LABELS: Record<QueryPlan["intent"], string> = {
  technology_search: "подбор технологии",
  experiment_search: "поиск экспериментов",
  literature_review: "обзор литературы",
  expert_search: "поиск экспертов",
  gap_analysis: "анализ пробелов",
  contradiction_analysis: "анализ противоречий",
  comparison: "сравнение",
  entity_lookup: "карточка сущности",
};

export function PlanChip({ plan }: { plan: QueryPlan }) {
  const entities = [
    ...plan.entities.materials,
    ...plan.entities.processes,
    ...plan.entities.properties,
  ];

  return (
    <div className="rise-in rounded-sm border border-line-strong bg-bg-1 px-4 py-3">
      <div className="flex items-center gap-3">
        <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          intent
        </span>
        <span className="text-[13px] font-semibold text-ink-0">
          {INTENT_LABELS[plan.intent]}
        </span>
        <span className="ml-auto flex items-center gap-2 font-mono text-[10px] text-ink-2">
          <span
            className="rounded-sm border border-line px-1.5 py-0.5 uppercase"
            title={
              plan.parser === "llm"
                ? "разбор запроса: LLM"
                : "разбор запроса: правила"
            }
          >
            {plan.parser}
          </span>
          <span>{Math.round(plan.confidence * 100)}%</span>
        </span>
      </div>
      <div className="mt-2 flex flex-wrap gap-1.5">
        {entities.map((entity, index) => (
          <Link
            key={`${entity.slug}-${index}`}
            href={`/entity/${encodeURIComponent(entity.slug)}`}
            title="Открыть паспорт сущности"
            className="inline-flex items-center gap-1.5 rounded-sm border border-electrolyte/30 bg-bg-2 px-2 py-1 text-[12px] text-ink-0 transition-colors hover:border-electrolyte"
          >
            <span className="font-mono text-[9px] uppercase text-ink-2">
              {slugPrefix(entity.slug)}
            </span>
            {entity.name}
          </Link>
        ))}
        {plan.paramConstraints.map((constraint, index) => (
          <span
            key={`${constraint.parameter.slug}-${index}`}
            className="inline-flex items-center gap-1.5 rounded-sm border border-line bg-bg-2 px-2 py-1 text-[11px] text-ink-1"
          >
            <span className="font-mono">{constraint.parameter.name}</span>
            <FactValue value={constraint.value} className="text-[11px]" />
          </span>
        ))}
      </div>
    </div>
  );
}
