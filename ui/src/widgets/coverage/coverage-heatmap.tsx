"use client";

import Link from "next/link";
import { useState } from "react";
import {
  COVERAGE_CELLS,
  COVERAGE_MATERIALS,
  COVERAGE_PROCESSES,
  type CoverageCell,
} from "@/shared/api/mock/coverage-scenario";

export function CoverageHeatmap() {
  const [selected, setSelected] = useState<CoverageCell | null>(null);

  const cellFor = (material: string, process: string) =>
    COVERAGE_CELLS.find(
      (cell) => cell.material === material && cell.process === process,
    );

  return (
    <div className="flex flex-col gap-4 xl:flex-row">
      <div className="min-w-0 flex-1 overflow-x-auto">
        <table className="w-full border-collapse">
          <thead>
            <tr>
              <th className="p-1" />
              {COVERAGE_PROCESSES.map((process) => (
                <th
                  key={process}
                  className="p-1 pb-2 text-left font-mono text-[10px] font-normal uppercase tracking-wider text-ink-2"
                >
                  {process}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {COVERAGE_MATERIALS.map((material) => (
              <tr key={material}>
                <th className="max-w-32 p-1 pr-3 text-right font-mono text-[10px] font-normal text-ink-1">
                  {material}
                </th>
                {COVERAGE_PROCESSES.map((process) => {
                  const cell = cellFor(material, process);
                  if (!cell) return <td key={process} />;
                  const empty = cell.score === 0;
                  const active =
                    selected?.material === material &&
                    selected?.process === process;
                  return (
                    <td key={process} className="p-1">
                      <button
                        type="button"
                        onClick={() => setSelected(cell)}
                        title={`${material} × ${process}: ${cell.score}`}
                        className={`flex h-12 w-full min-w-20 items-center justify-center rounded-sm border font-mono text-[11px] tabular-nums transition-all ${
                          active
                            ? "border-focus"
                            : "border-line hover:border-line-strong"
                        } ${empty ? "hatch text-void" : "text-ink-0"}`}
                        style={
                          empty
                            ? undefined
                            : {
                                backgroundColor: `color-mix(in srgb, var(--electrolyte) ${Math.round(cell.score * 0.55)}%, var(--bg-1))`,
                              }
                        }
                      >
                        {empty ? "—" : cell.score}
                      </button>
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <aside className="w-full rounded-sm border border-line bg-bg-1 p-4 xl:w-[300px] xl:shrink-0">
        {selected ? (
          <CellDetails cell={selected} />
        ) : (
          <p className="text-[12px] text-ink-2">
            Кликните по ячейке — здесь появятся счётчики, причины пробела и
            переход к запросу
          </p>
        )}
      </aside>
    </div>
  );
}

function CellDetails({ cell }: { cell: CoverageCell }) {
  const question = `Что известно про ${cell.process} для материала «${cell.material}»?`;
  return (
    <div className="flex flex-col gap-3">
      <div>
        <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          ячейка
        </p>
        <p className="mt-1 text-[13px] font-semibold text-ink-0">
          {cell.material} × {cell.process}
        </p>
      </div>
      <div className="grid grid-cols-3 gap-2">
        <CellStat label="score" value={cell.score} />
        <CellStat label="фактов" value={cell.facts} />
        <CellStat label="эксп." value={cell.experiments} />
      </div>
      {cell.reasons.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {cell.reasons.map((reason) => (
            <span
              key={reason}
              className="rounded-sm border border-void/40 px-1.5 py-0.5 font-mono text-[10px] text-void"
            >
              {reason}
            </span>
          ))}
        </div>
      )}
      <Link
        href={`/?q=${encodeURIComponent(question)}`}
        className="mt-1 w-fit rounded-sm border border-electrolyte/40 px-3 py-1.5 text-[12px] text-electrolyte transition-colors hover:bg-electrolyte hover:text-bg-0"
      >
        сформировать запрос →
      </Link>
    </div>
  );
}

function CellStat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-sm border border-line bg-bg-0 px-2 py-1.5">
      <p className="font-mono text-[9px] uppercase text-ink-2">{label}</p>
      <p className="font-mono text-[14px] font-bold tabular-nums text-ink-0">
        {value}
      </p>
    </div>
  );
}
