"use client";

import { useState } from "react";
import { useAsk } from "@/features/ask/use-ask";
import type { Fact } from "@/shared/api/types";
import { PRESETS, type Preset } from "@/shared/config/presets";
import { IconGraph, IconSearch, IconStamp } from "@/shared/ui/icons";
import { AnswerFeed } from "./answer-feed";
import { Inspector } from "./inspector";
import { QueryPassport } from "./query-passport";

export function Workspace() {
  const { state, ask } = useAsk();
  const [input, setInput] = useState("");
  const [selectedFact, setSelectedFact] = useState<Fact | null>(null);

  const submit = (question: string) => {
    const trimmed = question.trim();
    if (!trimmed) return;
    setInput(trimmed);
    setSelectedFact(null);
    ask(trimmed);
  };

  return (
    <div className="glow-panel mx-auto flex max-w-[1440px] flex-col gap-6 px-6 py-8">
      <form
        onSubmit={(event) => {
          event.preventDefault();
          submit(input);
        }}
        className="flex gap-2"
      >
        <div className="flex h-12 flex-1 items-center gap-3 rounded-sm border border-line-strong bg-bg-1 px-4">
          <IconSearch className="text-ink-2" />
          <input
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder="Например: какая скорость циркуляции католита оптимальна при электроэкстракции никеля?"
            className="h-full flex-1 bg-transparent text-[14px] text-ink-0 placeholder:text-ink-2 focus:outline-none"
          />
        </div>
        <button
          type="submit"
          disabled={state.phase === "planning" || state.phase === "retrieving"}
          className="h-12 rounded-sm bg-electrolyte px-5 font-medium text-bg-0 transition-colors hover:bg-electrolyte/90 disabled:opacity-50"
        >
          Спросить
        </button>
      </form>

      {state.phase === "idle" ? (
        <EmptyState onAsk={submit} />
      ) : (
        <div className="flex flex-col gap-6 lg:flex-row">
          <QueryPassport plan={state.plan} onAsk={submit} />
          <AnswerFeed
            state={state}
            selectedFact={selectedFact}
            onSelectFact={setSelectedFact}
            onAsk={submit}
          />
          <aside className="w-full rounded-sm border border-line bg-bg-1 xl:w-[360px] xl:shrink-0">
            <Inspector fact={selectedFact} />
          </aside>
        </div>
      )}
    </div>
  );
}

function EmptyState({ onAsk }: { onAsk: (question: string) => void }) {
  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-8 py-4">
      <section className="rise-in">
        <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-electrolyte">
          Единая карта знаний R&D
        </p>
        <h1 className="mt-2 font-display text-3xl font-extrabold leading-tight text-ink-0">
          Что уже известно —<br />с числами и первоисточниками
        </h1>
        <p className="mt-3 max-w-xl text-[13px] text-ink-1">
          Задайте технический вопрос на естественном языке: материал, процесс,
          условия, география, период. Ответ — с доказательной базой, провенансом
          до цитаты и проверкой каждого числа.
        </p>
      </section>

      <section className="rise-in" style={{ animationDelay: "80ms" }}>
        <h2 className="mb-3 font-mono text-[11px] uppercase tracking-[0.2em] text-ink-2">
          Протоколы запросов · Q1–Q6 из ТЗ
        </h2>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {PRESETS.map((preset, index) => (
            <PresetCard
              key={preset.id}
              preset={preset}
              index={index}
              onAsk={onAsk}
            />
          ))}
        </div>
      </section>

      <section
        className="rise-in flex items-center gap-3 rounded-sm border border-line bg-bg-1 px-4 py-3 text-[12px] text-ink-2"
        style={{ animationDelay: "120ms" }}
      >
        <IconGraph className="shrink-0 text-void" />
        <p>
          Демо-режим: ответы собираются из mock-сценария, подключение к API —
          после готовности бэкенда.
        </p>
      </section>
    </div>
  );
}

const KIND_LABELS: Record<Preset["kind"], string> = {
  numeric: "числовой подбор",
  consensus: "консенсус",
  factual: "фактографический",
  comparison: "сравнение",
  gap: "пробелы",
  expert: "эксперты",
};

function PresetCard({
  preset,
  index,
  onAsk,
}: {
  preset: Preset;
  index: number;
  onAsk: (question: string) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onAsk(preset.question)}
      className="rise-in group flex flex-col gap-2 rounded-sm border border-line bg-bg-1 p-4 text-left transition-all hover:-translate-y-px hover:border-line-strong hover:bg-bg-2"
      style={{ animationDelay: `${120 + index * 40}ms` }}
    >
      <div className="flex w-full items-center justify-between">
        <span className="stamp-frame flex h-7 w-10 items-center justify-center bg-bg-0 font-mono text-[11px] font-bold text-electrolyte">
          {preset.code}
        </span>
        <span className="font-mono text-[10px] uppercase tracking-wider text-ink-2">
          {KIND_LABELS[preset.kind]}
        </span>
      </div>
      <h3 className="text-[13px] font-semibold leading-snug text-ink-0">
        {preset.title}
      </h3>
      <p className="line-clamp-3 text-[12px] leading-relaxed text-ink-1">
        {preset.question}
      </p>
      <span className="mt-auto flex items-center gap-1 pt-1 font-mono text-[10px] text-ink-2 transition-colors group-hover:text-electrolyte">
        <IconStamp width={12} height={12} />
        выполнить запрос
      </span>
    </button>
  );
}
