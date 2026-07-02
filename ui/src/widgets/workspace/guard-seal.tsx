import type { GuardReport } from "@/shared/api/types";

export function GuardSeal({ guard }: { guard: GuardReport }) {
  const ok = !guard.degraded && guard.violations === 0;
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
          ? `${guard.numbersChecked}/${guard.numbersChecked} чисел сверены с источниками`
          : "экстрактивный режим: показаны только проверенные факты"}
      </span>
      <span className="ml-auto font-mono text-[10px] uppercase tracking-[0.2em] opacity-70">
        numeric guard
      </span>
    </div>
  );
}
