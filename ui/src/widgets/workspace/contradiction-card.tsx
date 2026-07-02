import type { Contradiction } from "@/shared/api/types";

const STATUS_LABELS: Record<Contradiction["status"], string> = {
  judge_confirmed: "подтверждено судьёй",
  expert_confirmed: "подтверждено экспертом",
  suspected: "подозрение",
};

export function ContradictionCard({
  contradiction,
}: {
  contradiction: Contradiction;
}) {
  return (
    <div className="rounded-sm border border-melt/30 bg-bg-1">
      <div className="grid grid-cols-[1fr_2px_1fr]">
        <Statement refLabel={contradiction.aFactRef} text={contradiction.aStatement} />
        <div className="bg-melt/60" />
        <Statement refLabel={contradiction.bFactRef} text={contradiction.bStatement} />
      </div>
      <div className="border-t border-melt/30 bg-bg-0 px-4 py-3">
        <div className="flex items-center gap-3">
          <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-melt">
            вердикт судьи
          </span>
          <span className="font-mono text-[10px] text-ink-2">
            {STATUS_LABELS[contradiction.status]} ·{" "}
            {Math.round(contradiction.confidence * 100)}%
          </span>
        </div>
        <p className="mt-1 text-[12px] text-ink-1">{contradiction.cause}</p>
        <div className="mt-2 flex flex-wrap items-center gap-1.5">
          <span className="font-mono text-[10px] text-ink-2">конфаундеры:</span>
          {contradiction.confounders.map((confounder) => (
            <span
              key={confounder}
              className="rounded-sm border border-line bg-bg-1 px-1.5 py-0.5 font-mono text-[10px] text-ink-1"
            >
              {confounder}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}

function Statement({ refLabel, text }: { refLabel: string; text: string }) {
  return (
    <div className="px-4 py-3">
      <span className="font-mono text-[11px] font-bold text-electrolyte">
        {refLabel}
      </span>
      <p className="mt-1 text-[13px] leading-snug text-ink-0">{text}</p>
    </div>
  );
}
