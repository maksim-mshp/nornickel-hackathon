"use client";

import { useEffect, useMemo, useState } from "react";
import { getExperiments, type ExperimentRow } from "@/shared/api/browse";
import { toCsv } from "@/shared/lib/csv";
import { downloadFile } from "@/shared/lib/download";
import { pluralCount } from "@/shared/lib/plural";
import { useRole } from "@/shared/lib/role";

export default function ExperimentsPage() {
  const [rows, setRows] = useState<ExperimentRow[]>([]);
  const [process, setProcess] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [compare, setCompare] = useState(false);
  const roleToken = useRole((store) => store.token);

  useEffect(() => {
    let alive = true;
    getExperiments().then((data) => {
      if (alive) setRows(data);
    });
    return () => {
      alive = false;
    };
  }, [roleToken]);

  const processes = useMemo(
    () => Array.from(new Set(rows.map((row) => row.process))),
    [rows],
  );
  const visible = useMemo(
    () => (process ? rows.filter((row) => row.process === process) : rows),
    [rows, process],
  );
  const selectedRows = visible.filter((row) => selected.has(row.id));

  useEffect(() => {
    if (compare && selectedRows.length < 2) setCompare(false);
  }, [compare, selectedRows.length]);

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else if (next.size < 4) next.add(id);
      return next;
    });
  };

  const exportCsv = () => {
    const header = ["code", "material", "process", "result", "source", "confidence"];
    const body = visible.map((row) => [
      row.code,
      row.material,
      row.process,
      row.result,
      row.source,
      row.confidence,
    ]);
    downloadFile("experiments.csv", toCsv([header, ...body]), "text/csv");
  };

  return (
    <div className="mx-auto flex max-w-[1280px] gap-6 px-6 py-8">
      <aside className="w-48 shrink-0">
        <h2 className="mb-2 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          процесс
        </h2>
        <div className="flex flex-col gap-1">
          <FacetButton active={process === null} onClick={() => setProcess(null)}>
            все
          </FacetButton>
          {processes.map((item) => (
            <FacetButton
              key={item}
              active={process === item}
              onClick={() => setProcess(item)}
            >
              {item}
            </FacetButton>
          ))}
        </div>
      </aside>

      <div className="min-w-0 flex-1">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h1 className="font-display text-xl font-extrabold text-ink-0">
              Каталог экспериментов
            </h1>
            <p className="mt-1 text-[13px] text-ink-1">
              {pluralCount(visible.length, "серия", "серии", "серий")} · условия
              и результаты со ссылкой на источник
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setCompare((value) => !value)}
              disabled={selectedRows.length < 2}
              className={`h-9 rounded-sm border px-3 font-mono text-[11px] transition-colors disabled:opacity-40 ${
                compare
                  ? "border-electrolyte text-electrolyte"
                  : "border-line text-ink-1 hover:border-line-strong"
              }`}
            >
              сравнить ({selectedRows.length})
            </button>
            <button
              type="button"
              onClick={exportCsv}
              className="h-9 rounded-sm border border-line px-3 font-mono text-[11px] text-ink-1 transition-colors hover:border-electrolyte hover:text-electrolyte"
            >
              экспорт CSV
            </button>
          </div>
        </div>

        {compare && selectedRows.length >= 2 ? (
          <CompareView rows={selectedRows} />
        ) : (
          <div className="overflow-x-auto rounded-sm border border-line">
            <table className="w-full border-collapse text-[13px]">
              <thead>
                <tr className="border-b border-line bg-bg-1 text-left font-mono text-[10px] uppercase tracking-wider text-ink-2">
                  <th className="w-8 px-2 py-2"></th>
                  <th className="px-2 py-2">ID</th>
                  <th className="px-2 py-2">материал</th>
                  <th className="px-2 py-2">условия</th>
                  <th className="px-2 py-2">результат</th>
                  <th className="px-2 py-2">источник</th>
                  <th className="px-2 py-2 text-right">conf.</th>
                </tr>
              </thead>
              <tbody>
                {visible.map((row, index) => (
                  <tr
                    key={row.id}
                    className={`border-b border-line/60 ${index % 2 ? "bg-bg-1/40" : ""}`}
                  >
                    <td className="px-2 py-2">
                      <input
                        type="checkbox"
                        checked={selected.has(row.id)}
                        onChange={() => toggle(row.id)}
                        className="accent-electrolyte"
                      />
                    </td>
                    <td className="px-2 py-2 font-mono text-[12px] font-bold text-electrolyte">
                      {row.code}
                    </td>
                    <td className="px-2 py-2 text-ink-0">{row.material}</td>
                    <td className="px-2 py-2">
                      <div className="flex flex-wrap gap-1">
                        {Object.entries(row.conditions).map(([key, value]) => (
                          <span
                            key={key}
                            className="rounded-sm border border-line px-1 font-mono text-[10px] text-ink-1"
                          >
                            {key}: {value}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="px-2 py-2 text-ink-1">{row.result}</td>
                    <td className="px-2 py-2 text-[12px] text-ink-2">{row.source}</td>
                    <td className="px-2 py-2 text-right font-mono tabular-nums text-ink-1">
                      {row.confidence.toFixed(2)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

function FacetButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-sm px-2 py-1.5 text-left text-[12px] transition-colors ${
        active ? "bg-bg-2 text-ink-0" : "text-ink-2 hover:text-ink-1"
      }`}
    >
      {children}
    </button>
  );
}

function CompareView({ rows }: { rows: ExperimentRow[] }) {
  const conditionKeys = Array.from(
    new Set(rows.flatMap((row) => Object.keys(row.conditions))),
  );
  return (
    <div className="overflow-x-auto rounded-sm border border-line">
      <table className="w-full border-collapse text-[13px]">
        <thead>
          <tr className="border-b border-line bg-bg-1 text-left">
            <th className="px-3 py-2 font-mono text-[10px] uppercase text-ink-2">
              параметр
            </th>
            {rows.map((row) => (
              <th
                key={row.id}
                className="px-3 py-2 font-mono text-[12px] font-bold text-electrolyte"
              >
                {row.code}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          <CompareRow label="материал" values={rows.map((row) => row.material)} />
          <CompareRow label="процесс" values={rows.map((row) => row.process)} />
          {conditionKeys.map((key) => (
            <CompareRow
              key={key}
              label={key}
              values={rows.map((row) => row.conditions[key] ?? "—")}
            />
          ))}
          <CompareRow label="результат" values={rows.map((row) => row.result)} />
        </tbody>
      </table>
    </div>
  );
}

function CompareRow({ label, values }: { label: string; values: string[] }) {
  const distinct = new Set(values).size > 1;
  return (
    <tr className="border-b border-line/60">
      <td className="px-3 py-2 font-mono text-[11px] text-ink-2">{label}</td>
      {values.map((value, index) => (
        <td
          key={`${label}-${index}`}
          className={`px-3 py-2 ${distinct ? "text-melt" : "text-ink-0"}`}
        >
          {value}
        </td>
      ))}
    </tr>
  );
}
