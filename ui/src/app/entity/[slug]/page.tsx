"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { getEntity } from "@/shared/api/mock/entity-scenario";
import { ConsensusSpectrum } from "@/widgets/consensus-spectrum/consensus-spectrum";
import { ExpertsList } from "@/widgets/workspace/gaps-experts";

const nf = new Intl.NumberFormat("ru-RU");

export default function EntityPage() {
  const params = useParams<{ slug: string }>();
  const entity = getEntity(decodeURIComponent(params.slug));

  if (!entity) {
    return (
      <div className="mx-auto max-w-3xl px-6 py-16 text-center">
        <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-ink-2">
          сущность не найдена
        </p>
        <p className="mt-3 text-[13px] text-ink-1">
          Паспорт этой сущности появится после индексации корпуса
        </p>
      </div>
    );
  }

  return (
    <div className="mx-auto flex max-w-[1200px] flex-col gap-8 px-6 py-8">
      <section className="rise-in flex flex-wrap items-start gap-6">
        <div className="min-w-0 flex-1">
          <span className="rounded-sm border border-electrolyte/40 px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider text-electrolyte">
            {entity.type}
          </span>
          <h1 className="mt-2 font-display text-2xl font-extrabold text-ink-0">
            {entity.nameRu}
          </h1>
          <p className="font-mono text-[12px] text-ink-2">{entity.nameEn}</p>
          <div className="mt-3 flex flex-wrap gap-1.5">
            {entity.synonyms.map((synonym) => (
              <span
                key={synonym.value}
                className={`rounded-sm border px-2 py-0.5 text-[11px] ${
                  synonym.pending
                    ? "border-line text-ink-2"
                    : "border-line-strong text-ink-1"
                }`}
                title={synonym.pending ? "ожидает подтверждения" : undefined}
              >
                {synonym.value}
              </span>
            ))}
          </div>
        </div>
        <div className="flex gap-4">
          {(
            [
              ["документы", entity.counters.documents],
              ["факты", entity.counters.facts],
              ["эксперименты", entity.counters.experiments],
              ["эксперты", entity.counters.experts],
            ] as const
          ).map(([label, value]) => (
            <div key={label} className="text-right">
              <p className="font-display text-3xl font-extrabold tabular-nums text-ink-0">
                {nf.format(value)}
              </p>
              <p className="font-mono text-[9px] uppercase tracking-wider text-ink-2">
                {label}
              </p>
            </div>
          ))}
        </div>
      </section>

      <Link
        href={`/?q=${encodeURIComponent(`Что известно про ${entity.nameRu}?`)}`}
        className="rise-in w-fit rounded-sm bg-electrolyte px-4 py-2 text-[13px] font-medium text-bg-0 transition-colors hover:bg-electrolyte/90"
        style={{ animationDelay: "40ms" }}
      >
        Спросить об этом →
      </Link>

      <div
        className="rise-in grid grid-cols-1 gap-6 xl:grid-cols-[1fr_320px_320px]"
        style={{ animationDelay: "80ms" }}
      >
        <section>
          <h2 className="mb-3 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
            Параметры · консенсус
          </h2>
          <div className="flex flex-col gap-3">
            {entity.consensus.map((consensus) => (
              <ConsensusSpectrum
                key={consensus.parameter.slug}
                consensus={consensus}
              />
            ))}
          </div>
        </section>
        <section>
          <h2 className="mb-3 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
            Связи
          </h2>
          <div className="flex flex-col gap-4 rounded-sm border border-line bg-bg-1 p-4">
            {entity.relations.map((relation) => (
              <div key={relation.group}>
                <p className="font-mono text-[10px] text-ink-2">
                  {relation.group}
                </p>
                <div className="mt-1.5 flex flex-col gap-1.5">
                  {relation.items.map((item) => (
                    <div key={item.slug} className="flex items-center gap-2">
                      <span
                        className="h-0.5 shrink-0 bg-electrolyte"
                        style={{ width: `${8 + item.weight * 24}px` }}
                      />
                      <span className="truncate text-[12px] text-ink-0">
                        {item.name}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </section>
        <section>
          <h2 className="mb-3 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
            Люди
          </h2>
          <ExpertsList experts={entity.experts} />
        </section>
      </div>

      <section className="rise-in" style={{ animationDelay: "120ms" }}>
        <h2 className="mb-3 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          Хронология фактов
        </h2>
        <div className="flex items-end gap-2 rounded-sm border border-line bg-bg-1 p-4">
          {entity.timeline.map((point) => {
            const max = Math.max(...entity.timeline.map((p) => p.facts));
            return (
              <div
                key={point.year}
                className="flex flex-1 flex-col items-center gap-1"
                title={`${point.year}: ${point.facts} фактов`}
              >
                <span className="font-mono text-[9px] tabular-nums text-ink-2">
                  {point.facts}
                </span>
                <span
                  className="w-full max-w-10 rounded-sm bg-electrolyte/70"
                  style={{ height: `${8 + (point.facts / max) * 72}px` }}
                />
                <span className="font-mono text-[9px] text-ink-2">
                  {point.year}
                </span>
              </div>
            );
          })}
        </div>
      </section>
    </div>
  );
}
