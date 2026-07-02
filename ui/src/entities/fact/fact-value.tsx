import type { NumericValue } from "@/shared/api/types";

const nf = new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 4 });

const PREFIX: Partial<Record<NumericValue["operator"], string>> = {
  lt: "<",
  lte: "≤",
  gt: ">",
  gte: "≥",
  approx: "≈",
  from: "от",
  to: "до",
};

export function formatFactValue(value: NumericValue): string {
  const min = value.vmin !== undefined ? nf.format(value.vmin) : "";
  const max = value.vmax !== undefined ? nf.format(value.vmax) : "";
  if (value.operator === "range") return `${min}–${max}`;
  if (value.operator === "pm") return `${min} ± ${max}`;
  const prefix = PREFIX[value.operator];
  const body = min || max;
  return prefix ? `${prefix} ${body}` : body;
}

export function FactValue({
  value,
  className = "",
}: {
  value: NumericValue;
  className?: string;
}) {
  const min = value.vmin !== undefined ? nf.format(value.vmin) : "";
  const max = value.vmax !== undefined ? nf.format(value.vmax) : "";
  const prefix = PREFIX[value.operator];

  return (
    <span
      className={`inline-flex items-baseline gap-1 whitespace-nowrap font-mono tabular-nums ${className}`}
    >
      {prefix && (
        <span className="font-normal text-electrolyte">{prefix}</span>
      )}
      <span className="font-bold text-ink-0">
        {value.operator === "range" ? (
          <>
            {min}
            <span className="font-normal text-electrolyte">–</span>
            {max}
          </>
        ) : value.operator === "pm" ? (
          <>
            {min}
            <span className="font-normal text-electrolyte"> ± </span>
            {max}
          </>
        ) : (
          min || max
        )}
      </span>
      <span className="font-normal text-ink-1">{value.unit}</span>
    </span>
  );
}
