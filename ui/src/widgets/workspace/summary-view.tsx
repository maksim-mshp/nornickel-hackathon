"use client";

import { Fragment } from "react";
import type { AnswerDoc } from "@/shared/api/types";

export function SummaryText({
  text,
  streaming,
  onRefClick,
}: {
  text: string;
  streaming: boolean;
  onRefClick: (ref: string) => void;
}) {
  const parts = text.split(/\[(F\d+)\]/g);
  return (
    <p
      className="text-[14px] leading-relaxed text-ink-0"
      aria-live={streaming ? "off" : "polite"}
    >
      {parts.map((part, index) =>
        index % 2 === 1 ? (
          <button
            key={index}
            type="button"
            onClick={() => onRefClick(part)}
            className="mx-0.5 rounded-sm border border-electrolyte/40 bg-bg-2 px-1 font-mono text-[11px] font-bold text-electrolyte transition-colors hover:bg-electrolyte hover:text-bg-0"
          >
            {part}
          </button>
        ) : (
          <Fragment key={index}>{part}</Fragment>
        ),
      )}
      {streaming && (
        <span className="ml-0.5 inline-block h-4 w-2 animate-pulse bg-electrolyte align-text-bottom" />
      )}
    </p>
  );
}

export function MethodsList({
  answer,
  onRefClick,
}: {
  answer: AnswerDoc;
  onRefClick: (ref: string) => void;
}) {
  return (
    <div className="mt-4 flex flex-col gap-2">
      <h3 className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
        Методы и применимость
      </h3>
      {answer.methods.map((method) => (
        <div
          key={method.name}
          className="flex flex-wrap items-baseline gap-x-3 gap-y-1 rounded-sm border border-line bg-bg-1 px-3 py-2"
        >
          <span className="text-[13px] font-semibold text-ink-0">
            {method.name}
          </span>
          <span className="text-[12px] text-ink-1">{method.applicability}</span>
          <span className="ml-auto flex gap-1">
            {method.citations.map((ref) => (
              <button
                key={ref}
                type="button"
                onClick={() => onRefClick(ref)}
                className="rounded-sm border border-electrolyte/40 px-1 font-mono text-[10px] font-bold text-electrolyte transition-colors hover:bg-electrolyte hover:text-bg-0"
              >
                {ref}
              </button>
            ))}
          </span>
        </div>
      ))}
    </div>
  );
}
