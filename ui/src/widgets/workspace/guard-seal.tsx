import type { GuardReport } from "@/shared/api/types";
import { pluralCount } from "@/shared/lib/plural";

export function GuardSeal({ guard }: { guard: GuardReport }) {
  const total = Math.max(0, guard.numbersChecked);
  const violations = Math.min(Math.max(0, guard.violations), total);
  const ok = !guard.degraded && violations === 0;
  const verified = total - violations;
  return (
    <div
      className={`seal-snap flex items-center gap-3 border border-dashed px-4 py-2.5 ${
        ok
          ? "border-electrolyte/50 bg-electrolyte/5 text-electrolyte"
          : "border-anode/60 bg-anode/5 text-anode"
      }`}
    >
      <span className="font-mono text-base leading-none">{ok ? "☑" : "⚠"}</span>
      <span className="font-mono text-[12px] tabular-nums">
        {ok
          ? `${verified}/${total} чисел сверены с источниками`
          : `${verified}/${total} сверены · ${pluralCount(violations, "расхождение", "расхождения", "расхождений")} — экстрактивный режим`}
      </span>
      <span className="ml-auto font-mono text-[10px] uppercase tracking-[0.2em] opacity-70">
        numeric guard
      </span>
    </div>
  );
}
