"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { getExperts, type ExpertProfile } from "@/shared/api/browse";
import { Isolines } from "@/shared/ui/isolines";

const CURRENT_YEAR = 2026;

export default function ExpertsPage() {
  const [experts, setExperts] = useState<ExpertProfile[]>([]);
  const [query, setQuery] = useState("");
  const [activeOnly, setActiveOnly] = useState(false);

  useEffect(() => {
    let alive = true;
    getExperts().then((data) => {
      if (alive) setExperts(data);
    });
    return () => {
      alive = false;
    };
  }, []);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return experts.filter((expert) => {
      if (activeOnly && CURRENT_YEAR - expert.lastYear > 3) return false;
      if (!needle) return true;
      return (
        expert.name.toLowerCase().includes(needle) ||
        expert.lab.toLowerCase().includes(needle) ||
        expert.topics.some((topic) => topic.toLowerCase().includes(needle))
      );
    });
  }, [experts, query, activeOnly]);

  return (
    <div className="mx-auto flex max-w-[1200px] flex-col gap-6 px-6 py-8">
      <section className="rise-in relative">
        <Isolines />
        <h1 className="font-display text-xl font-extrabold text-ink-0">
          Институциональная память
        </h1>
        <p className="mt-1 text-[13px] text-ink-1">
          Кто работал над темой, с какими экспериментами и отчётами —
          доказательная цепочка до первоисточника
        </p>
      </section>

      <section
        className="rise-in flex flex-wrap items-center gap-3"
        style={{ animationDelay: "40ms" }}
      >
        <div className="flex h-10 flex-1 items-center gap-2 rounded-sm border border-line bg-bg-1 px-3">
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Тема, сущность или ФИО: электроэкстракция, циркуляция католита…"
            className="h-full flex-1 bg-transparent text-[13px] text-ink-0 placeholder:text-ink-2 focus:outline-none"
          />
        </div>
        <button
          type="button"
          onClick={() => setActiveOnly((value) => !value)}
          className={`h-10 rounded-sm border px-3 font-mono text-[11px] transition-colors ${
            activeOnly
              ? "border-electrolyte text-electrolyte"
              : "border-line text-ink-2 hover:text-ink-1"
          }`}
        >
          только активные ≤3 лет
        </button>
      </section>

      <section className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {filtered.map((expert, index) => (
          <ExpertCard key={expert.id} expert={expert} index={index} />
        ))}
        {filtered.length === 0 && (
          <p className="col-span-full py-10 text-center text-[13px] text-ink-2">
            Экспертов по запросу не найдено
          </p>
        )}
      </section>
    </div>
  );
}

function ExpertCard({ expert, index }: { expert: ExpertProfile; index: number }) {
  const initials = expert.name
    .split(" ")
    .slice(0, 2)
    .map((part) => part[0])
    .join("");

  return (
    <article
      className="rise-in flex flex-col gap-4 rounded-sm border border-line bg-bg-1 p-5"
      style={{ animationDelay: `${index * 40}ms` }}
    >
      <div className="flex items-start gap-4">
        <span className="stamp-frame flex h-12 w-12 shrink-0 items-center justify-center bg-bg-0 font-mono text-[15px] font-bold text-anode">
          {initials}
        </span>
        <div className="min-w-0 flex-1">
          <h2 className="text-[15px] font-semibold text-ink-0">{expert.name}</h2>
          <p className="text-[12px] text-ink-2">{expert.lab}</p>
          <div className="mt-1.5 flex flex-wrap gap-1.5">
            {expert.topics.map((topic) => (
              <span
                key={topic}
                className="rounded-sm border border-line-strong px-1.5 py-0.5 font-mono text-[10px] text-ink-1"
              >
                {topic}
              </span>
            ))}
          </div>
        </div>
        <Activity activity={expert.activity} />
      </div>

      <div className="flex items-center gap-3">
        <span className="font-mono text-[10px] uppercase tracking-wider text-ink-2">
          вес по теме
        </span>
        <span className="h-1.5 flex-1 overflow-hidden rounded-sm bg-bg-2">
          <span
            className="block h-full bg-electrolyte"
            style={{ width: `${expert.weight * 100}%` }}
          />
        </span>
        <span className="font-mono text-[11px] tabular-nums text-electrolyte">
          {expert.weight.toFixed(2)}
        </span>
      </div>

      <div className="flex gap-4 font-mono text-[11px] text-ink-2">
        <span>
          отчётов <span className="tabular-nums text-ink-0">{expert.reports}</span>
        </span>
        <span>
          экспериментов{" "}
          <span className="tabular-nums text-ink-0">{expert.experiments}</span>
        </span>
        <span>
          активность <span className="tabular-nums text-ink-0">{expert.lastYear}</span>
        </span>
      </div>

      <div className="border-t border-line pt-3">
        <p className="mb-2 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          доказательная цепочка
        </p>
        <div className="flex flex-col gap-1.5">
          {expert.evidence.map((item) => (
            <div key={item.title} className="flex items-center gap-2 text-[12px]">
              <span className="font-mono text-[9px] uppercase text-ink-2">
                {item.kind}
              </span>
              <span className="truncate text-ink-1">{item.title}</span>
              <span className="ml-auto font-mono text-[10px] tabular-nums text-ink-2">
                {item.year}
              </span>
            </div>
          ))}
        </div>
      </div>

      <Link
        href={`/?q=${encodeURIComponent(`Кто работал над темой ${expert.topics[0] ?? expert.name}?`)}`}
        className="w-fit font-mono text-[11px] text-electrolyte transition-colors hover:underline"
      >
        показать в графе →
      </Link>
    </article>
  );
}

function Activity({ activity }: { activity: { year: number; count: number }[] }) {
  const max = Math.max(...activity.map((point) => point.count), 1);
  return (
    <div className="flex items-end gap-0.5" title="активность по годам">
      {activity.map((point) => (
        <span
          key={point.year}
          className="w-1.5 rounded-sm bg-electrolyte/60"
          style={{ height: `${6 + (point.count / max) * 26}px` }}
        />
      ))}
    </div>
  );
}
