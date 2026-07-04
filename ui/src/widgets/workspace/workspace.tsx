"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAsk } from "@/features/ask/use-ask";
import type { Fact } from "@/shared/api/types";
import { PRESETS } from "@/shared/config/presets";
import { IconSearch } from "@/shared/ui/icons";
import { Isolines } from "@/shared/ui/isolines";
import { AnswerFeed } from "./answer-feed";
import { Inspector } from "./inspector";
import { QueryPassport } from "./query-passport";

export function Workspace() {
  const { state, ask, reset } = useAsk();
  const [input, setInput] = useState("");
  const [selectedFact, setSelectedFact] = useState<Fact | null>(null);
  const lastAsked = useRef("");

  const runQuestion = useCallback(
    (question: string, force = false) => {
      const trimmed = question.trim();
      if (!trimmed || (!force && trimmed === lastAsked.current)) return;
      lastAsked.current = trimmed;
      setInput(trimmed);
      setSelectedFact(null);
      window.history.replaceState(
        null,
        "",
        `/?q=${encodeURIComponent(trimmed)}`,
      );
      ask(trimmed);
    },
    [ask],
  );

  useEffect(() => {
    const q = new URLSearchParams(window.location.search).get("q");
    if (q) runQuestion(q);
    const onAskEvent = (event: Event) => {
      runQuestion((event as CustomEvent<string>).detail ?? "");
    };
    window.addEventListener("kmap:ask", onAskEvent);
    return () => window.removeEventListener("kmap:ask", onAskEvent);
  }, [runQuestion]);

  const submit = (question: string) => runQuestion(question, true);
  const newQuery = () => {
    lastAsked.current = "";
    setInput("");
    setSelectedFact(null);
    window.history.replaceState(null, "", "/");
    reset();
  };
  const busy =
    state.phase === "planning" ||
    state.phase === "retrieving" ||
    state.phase === "streaming";

  if (state.phase === "idle") {
    return (
      <Landing
        input={input}
        onInput={setInput}
        onSubmit={() => submit(input)}
        onAsk={submit}
        busy={busy}
      />
    );
  }

  return (
    <div className="glow-panel mx-auto flex max-w-[1440px] flex-col gap-6 px-6 py-8">
      <SearchForm
        input={input}
        onInput={setInput}
        onSubmit={() => submit(input)}
        onReset={newQuery}
        busy={busy}
      />
      <div className="flex flex-col gap-6 lg:flex-row">
        <QueryPassport plan={state.plan} onAsk={submit} />
        <AnswerFeed
          state={state}
          selectedFact={selectedFact}
          onSelectFact={setSelectedFact}
          onAsk={submit}
        />
        <aside className="w-full rounded-sm border border-line bg-bg-1 xl:w-[360px] xl:shrink-0">
          <Inspector fact={selectedFact} plan={state.plan} pack={state.pack} />
        </aside>
      </div>
    </div>
  );
}

function SearchForm({
  input,
  onInput,
  onSubmit,
  onReset,
  busy,
}: {
  input: string;
  onInput: (value: string) => void;
  onSubmit: () => void;
  onReset: () => void;
  busy: boolean;
}) {
  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
      className="flex gap-2"
    >
      <button
        type="button"
        onClick={onReset}
        title="Новый запрос"
        className="h-12 shrink-0 rounded-sm border border-line px-3 font-mono text-[12px] text-ink-2 transition-colors hover:border-electrolyte hover:text-electrolyte"
      >
        ← новый
      </button>
      <div className="flex h-12 flex-1 items-center gap-3 rounded-sm border border-line-strong bg-bg-1 px-4">
        <IconSearch className="text-ink-2" />
        <input
          name="q"
          aria-label="Поисковый запрос"
          value={input}
          onChange={(event) => onInput(event.target.value)}
          placeholder="Например: какая скорость циркуляции католита оптимальна при электроэкстракции никеля?"
          className="h-full flex-1 bg-transparent text-[14px] text-ink-0 placeholder:text-ink-2 focus:outline-none"
        />
      </div>
      <button
        type="submit"
        disabled={busy}
        className="h-12 rounded-sm bg-electrolyte px-5 font-medium text-bg-0 transition-colors hover:bg-electrolyte/90 disabled:opacity-50"
      >
        Спросить
      </button>
    </form>
  );
}

function Landing({
  input,
  onInput,
  onSubmit,
  onAsk,
  busy,
}: {
  input: string;
  onInput: (value: string) => void;
  onSubmit: () => void;
  onAsk: (question: string) => void;
  busy: boolean;
}) {
  return (
    <div className="relative flex min-h-[calc(100vh-56px)] flex-col items-center justify-center px-6">
      <Isolines />
      <div className="glow-panel flex w-full max-w-2xl flex-col items-center gap-8">
        <div className="rise-in flex flex-col items-center gap-2">
          <span className="font-display text-5xl font-extrabold tracking-tight text-ink-0">
            <span className="text-electrolyte">◆</span> kmap
          </span>
          <span className="font-mono text-[11px] uppercase tracking-[0.3em] text-ink-2">
            карта знаний R&D
          </span>
        </div>

        <form
          onSubmit={(event) => {
            event.preventDefault();
            onSubmit();
          }}
          className="rise-in w-full"
          style={{ animationDelay: "60ms" }}
        >
          <div className="flex h-14 w-full items-center gap-3 rounded-full border border-line-strong bg-bg-1 px-6 shadow-lg transition-colors focus-within:border-electrolyte">
            <IconSearch className="shrink-0 text-ink-2" />
            <input
              autoFocus
              name="q"
              aria-label="Поисковый запрос"
              value={input}
              onChange={(event) => onInput(event.target.value)}
              placeholder="Задайте технический вопрос…"
              className="h-full flex-1 bg-transparent text-[15px] text-ink-0 placeholder:text-ink-2 focus:outline-none"
            />
            <button
              type="submit"
              disabled={busy || input.trim() === ""}
              className="shrink-0 rounded-full bg-electrolyte px-5 py-1.5 text-[13px] font-medium text-bg-0 transition-colors hover:bg-electrolyte/90 disabled:opacity-40"
            >
              Спросить
            </button>
          </div>
        </form>

        <div
          className="rise-in flex flex-wrap items-center justify-center gap-2"
          style={{ animationDelay: "120ms" }}
        >
          {PRESETS.map((preset) => (
            <button
              key={preset.id}
              type="button"
              onClick={() => onAsk(preset.question)}
              title={preset.title}
              className="flex items-center gap-1.5 rounded-full border border-line px-3 py-1 text-[12px] text-ink-1 transition-colors hover:border-electrolyte hover:text-electrolyte"
            >
              <span className="font-mono text-[10px] font-bold text-electrolyte">
                {preset.code}
              </span>
              {preset.title}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
