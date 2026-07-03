"use client";

import { useEffect, useState } from "react";
import { getEntities, type EntitySummaryLive } from "@/shared/api/browse";

type UnitRow = { code: string; names: string; dimension: string; si: string; factor: string };
type SynonymRow = { canonical: string; aliases: { value: string; lang: string; status: string }[] };

const UNITS: UnitRow[] = [
  { code: "m_per_s", names: "м/с · m/s", dimension: "velocity", si: "m/s", factor: "1" },
  { code: "celsius", names: "°C · град", dimension: "temperature", si: "K", factor: "+273,15" },
  { code: "mg_per_dm3", names: "мг/дм³ · mg/dm3", dimension: "mass_concentration", si: "kg/m³", factor: "1e-3" },
  { code: "mg_per_l", names: "мг/л · mg/L", dimension: "mass_concentration", si: "kg/m³", factor: "1e-3" },
  { code: "a_per_m2", names: "А/м² · A/m2", dimension: "current_density", si: "A/m²", factor: "1" },
  { code: "percent", names: "% · мас.% · об.%", dimension: "ratio", si: "%", factor: "1" },
  { code: "mpa", names: "МПа · MPa", dimension: "pressure", si: "Pa", factor: "1e6" },
];

const SYNONYMS: SynonymRow[] = [
  {
    canonical: "электроэкстракция никеля",
    aliases: [
      { value: "electrowinning", lang: "en", status: "active" },
      { value: "электровыделение никеля", lang: "ru", status: "active" },
    ],
  },
  {
    canonical: "сухой остаток",
    aliases: [
      { value: "TDS", lang: "en", status: "active" },
      { value: "солесодержание", lang: "ru", status: "pending" },
    ],
  },
  {
    canonical: "печь взвешенной плавки",
    aliases: [{ value: "ПВП", lang: "ru", status: "active" }],
  },
];

type Parsed = { operator: string; vmin?: number; vmax?: number; unit: string } | null;

function parseNumcore(input: string): Parsed {
  const text = input.replace(/ /g, " ").trim();
  const num = "(-?\\d+(?:[.,]\\d+)?)";
  const unit = "([а-яё°%/·²³\\w.]+(?:/[а-яё°%\\w²³.]+)?)?";
  const norm = (value: string) => parseFloat(value.replace(",", "."));

  let match = text.match(new RegExp(`${num}\\s*[–—-]\\s*${num}\\s*${unit}`, "iu"));
  if (match) return { operator: "range", vmin: norm(match[1]), vmax: norm(match[2]), unit: (match[3] ?? "").trim() };

  match = text.match(new RegExp(`(≤|не более|до|<)\\s*${num}\\s*${unit}`, "iu"));
  if (match) return { operator: "lte", vmax: norm(match[2]), unit: (match[3] ?? "").trim() };

  match = text.match(new RegExp(`(≥|не менее|от|свыше|выше|>)\\s*${num}\\s*${unit}`, "iu"));
  if (match) return { operator: "gte", vmin: norm(match[2]), unit: (match[3] ?? "").trim() };

  match = text.match(new RegExp(`${num}\\s*±\\s*${num}\\s*${unit}`, "iu"));
  if (match) {
    const center = norm(match[1]);
    const delta = norm(match[2]);
    return { operator: "range", vmin: center - delta, vmax: center + delta, unit: (match[3] ?? "").trim() };
  }

  match = text.match(new RegExp(`${num}\\s*${unit}`, "iu"));
  if (match) return { operator: "eq", vmin: norm(match[1]), vmax: norm(match[1]), unit: (match[2] ?? "").trim() };

  return null;
}

export default function DictionariesPage() {
  const [test, setTest] = useState("0,8–1,0 м/с");
  const [entities, setEntities] = useState<EntitySummaryLive[]>([]);
  const parsed = parseNumcore(test);

  useEffect(() => {
    let alive = true;
    getEntities().then((items) => {
      if (alive) setEntities(items);
    });
    return () => {
      alive = false;
    };
  }, []);

  return (
    <div className="mx-auto flex max-w-[1200px] flex-col gap-6 px-6 py-8">
      <section className="rise-in">
        <h1 className="font-display text-xl font-extrabold text-ink-0">Словари</h1>
        <p className="mt-1 text-[13px] text-ink-1">
          Синонимы сущностей и реестр единиц измерения с живым тестом парсинга numcore
          {entities.length > 0 ? " · реестр сущностей из базы" : ""}
        </p>
      </section>

      {entities.length > 0 && (
        <section className="rise-in" style={{ animationDelay: "20ms" }}>
          <h2 className="mb-3 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
            Сущности базы · {entities.length}
          </h2>
          <div className="flex flex-wrap gap-1.5">
            {entities.map((entity) => (
              <span
                key={entity.id}
                title={entity.nameEn || entity.slug}
                className="flex items-center gap-1.5 rounded-sm border border-line-strong bg-bg-1 px-2 py-1 text-[12px] text-ink-0"
              >
                {entity.name}
                <span className="font-mono text-[9px] uppercase text-ink-2">
                  {entity.etype}
                </span>
              </span>
            ))}
          </div>
        </section>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <section className="rise-in" style={{ animationDelay: "40ms" }}>
          <h2 className="mb-3 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
            Синонимы
          </h2>
          <div className="flex flex-col gap-3">
            {SYNONYMS.map((row) => (
              <div key={row.canonical} className="rounded-sm border border-line bg-bg-1 p-3">
                <p className="text-[13px] font-semibold text-ink-0">{row.canonical}</p>
                <div className="mt-2 flex flex-wrap gap-1.5">
                  {row.aliases.map((alias) => (
                    <span
                      key={alias.value}
                      className={`flex items-center gap-1 rounded-sm border px-1.5 py-0.5 font-mono text-[11px] ${
                        alias.status === "pending"
                          ? "border-anode/40 text-anode"
                          : "border-line-strong text-ink-1"
                      }`}
                    >
                      {alias.value}
                      <span className="text-[9px] text-ink-2">{alias.lang}</span>
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </section>

        <section className="rise-in" style={{ animationDelay: "80ms" }}>
          <h2 className="mb-3 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
            Единицы
          </h2>
          <div className="overflow-x-auto rounded-sm border border-line">
            <table className="w-full border-collapse text-[12px]">
              <thead>
                <tr className="border-b border-line bg-bg-1 text-left font-mono text-[9px] uppercase tracking-wider text-ink-2">
                  <th className="px-2 py-1.5">код</th>
                  <th className="px-2 py-1.5">написания</th>
                  <th className="px-2 py-1.5">размерность</th>
                  <th className="px-2 py-1.5">SI</th>
                  <th className="px-2 py-1.5">фактор</th>
                </tr>
              </thead>
              <tbody>
                {UNITS.map((unit, index) => (
                  <tr key={unit.code} className={`border-b border-line/60 ${index % 2 ? "bg-bg-1/40" : ""}`}>
                    <td className="px-2 py-1.5 font-mono text-electrolyte">{unit.code}</td>
                    <td className="px-2 py-1.5 font-mono text-ink-0">{unit.names}</td>
                    <td className="px-2 py-1.5 text-ink-2">{unit.dimension}</td>
                    <td className="px-2 py-1.5 font-mono text-ink-1">{unit.si}</td>
                    <td className="px-2 py-1.5 font-mono tabular-nums text-ink-1">{unit.factor}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>

      <section className="rise-in rounded-sm border border-line bg-bg-1 p-4" style={{ animationDelay: "120ms" }}>
        <h2 className="mb-3 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          Тест numcore · распарсить пример
        </h2>
        <div className="flex flex-wrap items-center gap-3">
          <input
            name="numcore-test"
            aria-label="Тест numcore: распарсить пример"
            value={test}
            onChange={(event) => setTest(event.target.value)}
            placeholder="0,8–1,0 м/с · ≤1000 мг/дм³ · 65±2 °C · ≥95 %"
            className="h-10 min-w-[280px] flex-1 rounded-sm border border-line bg-bg-0 px-3 font-mono text-[13px] text-ink-0 focus:border-electrolyte focus:outline-none"
          />
          <div className="font-mono text-[12px]">
            {parsed ? (
              <div className="flex flex-wrap items-center gap-2">
                <Chip label="operator" value={parsed.operator} />
                {parsed.vmin !== undefined && <Chip label="vmin" value={String(parsed.vmin)} />}
                {parsed.vmax !== undefined && <Chip label="vmax" value={String(parsed.vmax)} />}
                <Chip label="unit" value={parsed.unit || "—"} />
              </div>
            ) : (
              <span className="text-melt">число не распознано</span>
            )}
          </div>
        </div>
      </section>
    </div>
  );
}

function Chip({ label, value }: { label: string; value: string }) {
  return (
    <span className="flex items-center gap-1 rounded-sm border border-electrolyte/40 bg-bg-2 px-1.5 py-0.5">
      <span className="text-[9px] uppercase text-ink-2">{label}</span>
      <span className="font-bold text-electrolyte">{value}</span>
    </span>
  );
}
