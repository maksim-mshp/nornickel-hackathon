"use client";

import { useEffect, useState } from "react";
import { FactValue } from "@/entities/fact/fact-value";
import {
  getEntities,
  parseQueryConstraints,
  type EntitySummaryLive,
  type ParsedConstraint,
} from "@/shared/api/browse";

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

export default function DictionariesPage() {
  const [test, setTest] = useState("скорость потока 0,8–1,0 м/с");
  const [entities, setEntities] = useState<EntitySummaryLive[]>([]);
  const [constraints, setConstraints] = useState<ParsedConstraint[]>([]);
  const [parsing, setParsing] = useState(false);

  useEffect(() => {
    let alive = true;
    getEntities().then((items) => {
      if (alive) setEntities(items);
    });
    return () => {
      alive = false;
    };
  }, []);

  useEffect(() => {
    let alive = true;
    const query = test.trim();
    if (!query) {
      setConstraints([]);
      setParsing(false);
      return;
    }
    setParsing(true);
    const timer = setTimeout(() => {
      void parseQueryConstraints(query).then((items) => {
        if (!alive) return;
        setConstraints(items);
        setParsing(false);
      });
    }, 400);
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [test]);

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
          Тест numcore · распарсить пример (детерминированный разбор на сервере)
        </h2>
        <div className="flex flex-wrap items-start gap-3">
          <input
            name="numcore-test"
            aria-label="Тест numcore: распарсить пример"
            value={test}
            onChange={(event) => setTest(event.target.value)}
            placeholder="скорость потока 0,8–1,0 м/с · сухой остаток ≤1000 мг/дм³ · температура 65 °C"
            className="h-10 min-w-[280px] flex-1 rounded-sm border border-line bg-bg-0 px-3 font-mono text-[13px] text-ink-0 focus:border-electrolyte focus:outline-none"
          />
          <div className="min-h-10 font-mono text-[12px]">
            {parsing ? (
              <span className="text-ink-2">разбор…</span>
            ) : constraints.length > 0 ? (
              <div className="flex flex-col gap-1.5">
                {constraints.map((constraint, index) => (
                  <div
                    key={`${constraint.parameter.slug}-${index}`}
                    className="flex flex-wrap items-center gap-2"
                  >
                    <span className="text-ink-1">{constraint.parameter.name}</span>
                    <FactValue value={constraint.value} className="text-[12px]" />
                  </div>
                ))}
              </div>
            ) : (
              <span className="text-melt">ограничение не распознано</span>
            )}
          </div>
        </div>
      </section>
    </div>
  );
}
