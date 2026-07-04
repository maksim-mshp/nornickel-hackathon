import type { NumericValue } from "@/shared/api/types";
import { formatNumber } from "@/shared/lib/format";

const PREFIX: Partial<Record<NumericValue["operator"], string>> = {
  lt: "<",
  lte: "≤",
  gt: ">",
  gte: "≥",
  approx: "≈",
  from: "от",
  to: "до",
};

function bounds(value: NumericValue) {
  const min = value.vmin !== undefined ? formatNumber(value.vmin) : "";
  const max = value.vmax !== undefined ? formatNumber(value.vmax) : "";
  return { min, max };
}

function isRange(value: NumericValue): boolean {
  return (
    (value.operator === "range" || value.operator === "pm") &&
    value.vmin !== undefined &&
    value.vmax !== undefined
  );
}

export function formatFactValue(value: NumericValue): string {
  const { min, max } = bounds(value);
  if (isRange(value)) {
    return value.operator === "pm" ? `${min} ± ${max}` : `${min}–${max}`;
  }
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
  const { min, max } = bounds(value);
  const prefix = PREFIX[value.operator];
  const range = isRange(value);

  return (
    <span
      className={`inline-flex items-baseline gap-1 whitespace-nowrap font-mono tabular-nums ${className}`}
    >
      {!range && prefix && (
        <span className="font-normal text-electrolyte">{prefix}</span>
      )}
      <span className="font-bold text-ink-0">
        {range ? (
          <>
            {min}
            <span className="font-normal text-electrolyte">
              {value.operator === "pm" ? " ± " : "–"}
            </span>
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
