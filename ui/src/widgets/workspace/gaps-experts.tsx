import type { Expert, GapCell } from "@/shared/api/types";

export function GapsList({
  gaps,
  onAsk,
}: {
  gaps: GapCell[];
  onAsk: (question: string) => void;
}) {
  if (gaps.length === 0) {
    return (
      <p className="rounded-sm border border-line bg-bg-1 p-4 text-[12px] text-ink-2">
        Пробелы по этому запросу не обнаружены.
      </p>
    );
  }
  return (
    <div className="flex flex-col gap-3">
      {gaps.map((gap, gapIndex) => {
        const neighbors = gap.neighbors ?? [];
        return (
          <div
            key={`${gap.label}-${gapIndex}`}
            className="rounded-sm border border-void/40 bg-bg-1 p-4"
          >
            <div className="flex items-center gap-3">
              <span className="hatch flex h-8 w-8 shrink-0 items-center justify-center rounded-sm border border-void/40 font-mono text-[10px] text-void">
                {gap.score}
              </span>
              <span className="text-[13px] font-semibold text-ink-0">
                {gap.label}
              </span>
            </div>
            <div className="mt-2 flex flex-wrap gap-1.5">
              {(gap.reasons ?? []).map((reason, reasonIndex) => (
                <span
                  key={`${reason}-${reasonIndex}`}
                  className="rounded-sm border border-void/40 px-1.5 py-0.5 font-mono text-[10px] text-void"
                >
                  {reason}
                </span>
              ))}
            </div>
            {neighbors.length > 0 && (
              <div className="mt-3 flex flex-col gap-1">
                <span className="font-mono text-[10px] uppercase tracking-wider text-ink-2">
                  смежные области
                </span>
                {neighbors.map((neighbor, neighborIndex) => (
                  <button
                    key={`${neighbor}-${neighborIndex}`}
                    type="button"
                    onClick={() => onAsk(neighbor)}
                    className="w-fit text-left text-[12px] text-electrolyte underline-offset-2 hover:underline"
                  >
                    → {neighbor}
                  </button>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

export function ExpertsList({ experts }: { experts: Expert[] }) {
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
      {experts.map((expert) => (
        <div
          key={expert.id}
          className="flex gap-3 rounded-sm border border-line bg-bg-1 p-4"
        >
          <span className="stamp-frame flex h-10 w-10 shrink-0 items-center justify-center bg-bg-0 font-display text-[13px] font-bold text-anode">
            {expert.name
              .split(" ")
              .filter(Boolean)
              .slice(0, 2)
              .map((part) => part[0] ?? "")
              .join("") || "—"}
          </span>
          <div className="min-w-0 flex-1">
            <p className="truncate text-[13px] font-semibold text-ink-0">
              {expert.name}
            </p>
            <p className="truncate text-[11px] text-ink-2">{expert.lab}</p>
            <div className="mt-2 flex items-center gap-2">
              <span className="h-1.5 w-20 overflow-hidden rounded-sm bg-bg-2">
                <span
                  className="block h-full bg-anode"
                  style={{ width: `${Math.round(expert.weight * 100)}%` }}
                />
              </span>
              <span className="font-mono text-[10px] tabular-nums text-ink-2">
                вес {expert.weight.toFixed(2)}
              </span>
            </div>
            <p className="mt-1 font-mono text-[10px] text-ink-2">
              {expert.reports} отч. · {expert.experiments} эксп. · активен{" "}
              {expert.lastYear}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}
