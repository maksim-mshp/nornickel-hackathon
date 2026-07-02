import type { QueryPlan } from "@/shared/api/types";
import { PRESETS } from "@/shared/config/presets";

const GEO_LABELS: Record<QueryPlan["geography"], string> = {
  any: "любая",
  ru: "Россия",
  foreign: "зарубежная",
  compare: "сравнить",
};

export function QueryPassport({
  plan,
  onAsk,
}: {
  plan: QueryPlan | null;
  onAsk: (question: string) => void;
}) {
  return (
    <aside className="flex w-full flex-col gap-4 lg:w-[264px] lg:shrink-0">
      <div className="rounded-sm border border-line bg-bg-1 p-4">
        <h2 className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          Паспорт запроса
        </h2>
        <div className="mt-3 flex flex-col gap-3">
          <PassportField
            label="материалы"
            values={plan?.entities.materials.map((e) => e.name)}
          />
          <PassportField
            label="процессы"
            values={plan?.entities.processes.map((e) => e.name)}
          />
          <PassportField
            label="параметры"
            values={plan?.entities.properties.map((e) => e.name)}
          />
          <PassportField
            label="география"
            values={plan ? [GEO_LABELS[plan.geography]] : undefined}
          />
          <PassportField
            label="период"
            values={
              plan
                ? [
                    plan.yearFrom || plan.yearTo
                      ? `${plan.yearFrom ?? "…"} — ${plan.yearTo ?? "…"}`
                      : "весь",
                  ]
                : undefined
            }
          />
        </div>
      </div>
      <div className="rounded-sm border border-line bg-bg-1 p-4">
        <h2 className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          Протоколы Q1–Q6
        </h2>
        <div className="mt-2 flex flex-col">
          {PRESETS.map((preset) => (
            <button
              key={preset.id}
              type="button"
              onClick={() => onAsk(preset.question)}
              className="group flex items-start gap-2 rounded-sm px-1 py-1.5 text-left transition-colors hover:bg-bg-2"
            >
              <span className="font-mono text-[10px] font-bold text-electrolyte">
                {preset.code}
              </span>
              <span className="text-[11px] leading-snug text-ink-1 group-hover:text-ink-0">
                {preset.title}
              </span>
            </button>
          ))}
        </div>
      </div>
    </aside>
  );
}

function PassportField({
  label,
  values,
}: {
  label: string;
  values?: string[];
}) {
  return (
    <div>
      <span className="font-mono text-[10px] text-ink-2">{label}</span>
      <div className="mt-1 flex min-h-6 flex-wrap gap-1">
        {values && values.length > 0 ? (
          values.map((value) => (
            <span
              key={value}
              className="rounded-sm border border-line bg-bg-0 px-1.5 py-0.5 text-[11px] text-ink-0"
            >
              {value}
            </span>
          ))
        ) : (
          <span className="hatch inline-block h-5 w-full rounded-sm opacity-60" />
        )}
      </div>
    </div>
  );
}
