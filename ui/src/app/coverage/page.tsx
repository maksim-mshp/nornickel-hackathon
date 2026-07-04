"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { getCoverageCells, type CoverageCellLive } from "@/shared/api/browse";
import {
  COVERAGE_CELLS,
  COVERAGE_KPIS,
  COVERAGE_MATERIALS,
  COVERAGE_PROCESSES,
  COVERAGE_RISKS,
  type CoverageKpi,
  type RiskItem,
} from "@/shared/api/mock/coverage-scenario";
import { useRole } from "@/shared/lib/role";
import { Isolines } from "@/shared/ui/isolines";
import { CoverageHeatmap } from "@/widgets/coverage/coverage-heatmap";

const nf = new Intl.NumberFormat("ru-RU");

const OUTDATED = new Set(["stale", "устарело"]);
const FOREIGN = new Set(["foreign_only", "no_ru_practice", "только зарубежные"]);
const CONTRADICTORY = new Set(["contradictory", "противоречия"]);

export default function CoveragePage() {
  const [live, setLive] = useState<CoverageCellLive[] | null>(null);
  const roleToken = useRole((store) => store.token);

  useEffect(() => {
    let alive = true;
    getCoverageCells().then((cells) => {
      if (alive) setLive(cells);
    });
    return () => {
      alive = false;
    };
  }, [roleToken]);

  const view = useMemo(() => buildView(live), [live]);

  return (
    <div className="mx-auto flex max-w-[1440px] flex-col gap-8 px-6 py-8">
      <section className="rise-in relative">
        <Isolines />
        <h1 className="font-display text-xl font-extrabold text-ink-0">
          Карта покрытия знаний
        </h1>
        <p className="mt-1 text-[13px] text-ink-1">
          Где база сильна, где противоречива, а где — пробелы
          {live ? " · данные из базы" : ""}
        </p>
      </section>

      <section
        className="rise-in grid grid-cols-2 gap-3 lg:grid-cols-4"
        style={{ animationDelay: "40ms" }}
      >
        {view.kpis.map((kpi) => (
          <KpiCard key={kpi.label} kpi={kpi} />
        ))}
      </section>

      <section className="rise-in" style={{ animationDelay: "80ms" }}>
        <h2 className="mb-3 font-mono text-[11px] uppercase tracking-[0.2em] text-ink-2">
          Heatmap · материал × процесс
        </h2>
        <CoverageHeatmap
          cells={view.cells}
          materials={view.materials}
          processes={view.processes}
        />
      </section>

      <section className="rise-in" style={{ animationDelay: "120ms" }}>
        <h2 className="mb-3 font-mono text-[11px] uppercase tracking-[0.2em] text-ink-2">
          Зоны риска
        </h2>
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
          <RiskColumn title="Противоречивые кластеры" tone="melt" items={view.risks.contradictory} />
          <RiskColumn title="Устаревшие темы" tone="anode" items={view.risks.outdated} />
          <RiskColumn title="Только зарубежное" tone="void" items={view.risks.foreignOnly} />
        </div>
      </section>
    </div>
  );
}

function buildView(live: CoverageCellLive[] | null) {
  if (!live || live.length === 0) {
    return {
      cells: COVERAGE_CELLS,
      materials: COVERAGE_MATERIALS,
      processes: COVERAGE_PROCESSES,
      kpis: COVERAGE_KPIS,
      risks: COVERAGE_RISKS,
    };
  }

  const materials = [...new Set(live.map((cell) => cell.material))].filter(Boolean);
  const processes = [...new Set(live.map((cell) => cell.process))].filter(Boolean);
  const totalFacts = live.reduce((sum, cell) => sum + cell.facts, 0);
  const totalExperiments = live.reduce((sum, cell) => sum + cell.experiments, 0);
  const gaps = live.filter((cell) => cell.reasons.length > 0).length;

  const kpis: CoverageKpi[] = [
    { label: "ячеек покрытия", value: live.length, trend: [live.length] },
    { label: "подтверждённых фактов", value: totalFacts, trend: [totalFacts] },
    { label: "экспериментов", value: totalExperiments, trend: [totalExperiments] },
    { label: "пробелов", value: gaps, trend: [gaps] },
  ];

  const risk = (predicate: (reasons: string[]) => boolean): RiskItem[] =>
    live
      .filter((cell) => predicate(cell.reasons))
      .map((cell) => ({
        label: `${cell.material} × ${cell.process}`,
        detail: cell.reasons.join(", "),
        question: `Что известно про ${cell.process} для материала «${cell.material}»?`,
      }));

  return {
    cells: live,
    materials,
    processes,
    kpis,
    risks: {
      contradictory: risk((reasons) => reasons.some((r) => CONTRADICTORY.has(r))),
      outdated: risk((reasons) => reasons.some((r) => OUTDATED.has(r))),
      foreignOnly: risk((reasons) => reasons.some((r) => FOREIGN.has(r))),
    },
  };
}

function KpiCard({ kpi }: { kpi: CoverageKpi }) {
  return (
    <div className="rounded-sm border border-line bg-bg-1 p-4">
      <p className="font-display text-4xl font-extrabold tabular-nums text-ink-0">
        {nf.format(kpi.value)}
      </p>
      <p className="mt-1 font-mono text-[10px] uppercase tracking-wider text-ink-2">
        {kpi.label}
      </p>
      {kpi.trend.length > 1 && <Sparkline points={kpi.trend} />}
    </div>
  );
}

function Sparkline({ points }: { points: number[] }) {
  const min = Math.min(...points);
  const max = Math.max(...points);
  const span = max - min || 1;
  const coords = points
    .map(
      (point, index) =>
        `${(index / (points.length - 1)) * 100},${24 - ((point - min) / span) * 20}`,
    )
    .join(" ");
  return (
    <svg viewBox="0 0 100 28" className="mt-2 h-7 w-full" aria-hidden>
      <polyline
        points={coords}
        fill="none"
        stroke="var(--electrolyte)"
        strokeWidth="1.5"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}

const TONE_CLASSES: Record<string, string> = {
  melt: "border-melt/40 text-melt",
  anode: "border-anode/40 text-anode",
  void: "border-void/40 text-void",
};

function RiskColumn({
  title,
  tone,
  items,
}: {
  title: string;
  tone: string;
  items: RiskItem[];
}) {
  return (
    <div className={`rounded-sm border bg-bg-1 p-4 ${TONE_CLASSES[tone]}`}>
      <h3 className="font-mono text-[10px] uppercase tracking-[0.2em]">{title}</h3>
      <div className="mt-3 flex flex-col gap-3">
        {items.length === 0 && (
          <p className="text-[11px] text-ink-2">нет</p>
        )}
        {items.map((item) => (
          <Link
            key={item.label}
            href={`/?q=${encodeURIComponent(item.question)}`}
            className="group block"
          >
            <p className="text-[13px] font-semibold text-ink-0 transition-colors group-hover:text-electrolyte">
              {item.label}
            </p>
            <p className="mt-0.5 text-[11px] text-ink-2">{item.detail}</p>
          </Link>
        ))}
      </div>
    </div>
  );
}
