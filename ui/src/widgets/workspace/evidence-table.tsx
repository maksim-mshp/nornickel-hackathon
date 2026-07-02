"use client";

import { FactValue } from "@/entities/fact/fact-value";
import {
  ProvenanceStamp,
  ValidationBadge,
} from "@/entities/fact/provenance-stamp";
import type { Fact } from "@/shared/api/types";

const GEO_LABELS: Record<string, string> = {
  ru: "РФ",
  foreign: "заруб.",
  global: "мир",
  unknown: "—",
};

export function EvidenceTable({
  facts,
  selectedId,
  onSelect,
}: {
  facts: Fact[];
  selectedId: string | null;
  onSelect: (fact: Fact) => void;
}) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse text-[12px]">
        <thead>
          <tr className="border-b border-line-strong text-left font-mono text-[10px] uppercase tracking-wider text-ink-2">
            <th className="px-2 py-2 font-normal">Ref</th>
            <th className="px-2 py-2 font-normal">Источник</th>
            <th className="px-2 py-2 font-normal">Параметр</th>
            <th className="px-2 py-2 font-normal">Значение</th>
            <th className="px-2 py-2 font-normal">Условия</th>
            <th className="px-2 py-2 font-normal">Гео</th>
            <th className="px-2 py-2 font-normal">Статус</th>
            <th className="px-2 py-2 font-normal">Score</th>
          </tr>
        </thead>
        <tbody>
          {facts.map((fact) => (
            <tr
              key={fact.id}
              onClick={() => onSelect(fact)}
              className={`cursor-pointer border-b border-line transition-colors ${
                selectedId === fact.id
                  ? "bg-bg-2"
                  : "hover:bg-bg-1"
              }`}
            >
              <td className="px-2 py-2 font-mono text-[11px] font-bold text-electrolyte">
                {fact.ref}
              </td>
              <td className="px-2 py-2">
                <ProvenanceStamp
                  provenance={fact.provenance}
                  method={fact.extractionMethod}
                  compact
                />
              </td>
              <td className="px-2 py-2 text-ink-1">{fact.parameter.name}</td>
              <td className="px-2 py-2">
                <FactValue value={fact.value} className="text-[12px]" />
              </td>
              <td className="px-2 py-2">
                <ConditionTags conditions={fact.conditions} />
              </td>
              <td className="px-2 py-2 font-mono text-[11px] text-ink-2">
                {GEO_LABELS[fact.geography]}
              </td>
              <td className="px-2 py-2">
                <ValidationBadge status={fact.validationStatus} />
              </td>
              <td className="px-2 py-2">
                <ScoreBar fact={fact} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ConditionTags({ conditions }: { conditions: Record<string, string> }) {
  const entries = Object.entries(conditions);
  if (entries.length === 0) {
    return <span className="hatch inline-block h-4 w-12 rounded-sm" />;
  }
  return (
    <span className="flex flex-wrap gap-1">
      {entries.map(([key, value]) => (
        <span
          key={key}
          className="whitespace-nowrap rounded-sm border border-line bg-bg-0 px-1.5 py-0.5 font-mono text-[10px] text-ink-1"
          title={key}
        >
          {value}
        </span>
      ))}
    </span>
  );
}

function ScoreBar({ fact }: { fact: Fact }) {
  const parts = Object.entries(fact.scoreComponents)
    .map(([key, value]) => `${key}: ${value.toFixed(2)}`)
    .join("\n");
  return (
    <span
      className="flex items-center gap-2"
      title={parts}
    >
      <span className="h-1.5 w-14 overflow-hidden rounded-sm bg-bg-2">
        <span
          className="block h-full bg-electrolyte"
          style={{ width: `${Math.round(fact.score * 100)}%` }}
        />
      </span>
      <span className="font-mono text-[10px] tabular-nums text-ink-2">
        {fact.score.toFixed(2)}
      </span>
    </span>
  );
}
