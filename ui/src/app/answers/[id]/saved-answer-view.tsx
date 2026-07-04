"use client";

import { notFound } from "next/navigation";
import { useEffect, useState } from "react";
import { FactValue } from "@/entities/fact/fact-value";
import {
  ProvenanceStamp,
  ValidationBadge,
} from "@/entities/fact/provenance-stamp";
import { getSavedAnswer, type SavedAnswer } from "@/shared/api/browse";
import {
  CATHOLYTE_ANSWER,
  CATHOLYTE_PACK,
} from "@/shared/api/mock/catholyte-scenario";
import { PRESETS } from "@/shared/config/presets";
import { GuardSeal } from "@/widgets/workspace/guard-seal";

const DEMO_SAVED: Record<string, SavedAnswer> = {
  q2: {
    question: PRESETS[1]?.question ?? "",
    answer: CATHOLYTE_ANSWER,
    pack: CATHOLYTE_PACK,
  },
};

type ViewState =
  | { status: "loading" }
  | { status: "found"; data: SavedAnswer }
  | { status: "missing" };

export function SavedAnswerView({ id }: { id: string }) {
  const [state, setState] = useState<ViewState>({ status: "loading" });

  useEffect(() => {
    let alive = true;
    setState({ status: "loading" });
    void getSavedAnswer(id).then((data) => {
      if (!alive) return;
      const resolved = data ?? DEMO_SAVED[id] ?? null;
      setState(resolved ? { status: "found", data: resolved } : { status: "missing" });
    });
    return () => {
      alive = false;
    };
  }, [id]);

  if (state.status === "missing") notFound();
  if (state.status === "loading") {
    return (
      <div className="mx-auto flex max-w-3xl flex-col gap-4 px-6 py-10">
        <div className="h-3 w-40 animate-pulse rounded-sm bg-bg-2" />
        <div className="h-6 w-3/4 animate-pulse rounded-sm bg-bg-2" />
        <div className="h-24 w-full animate-pulse rounded-sm bg-bg-1" />
        <div className="h-16 w-full animate-pulse rounded-sm bg-bg-1" />
      </div>
    );
  }

  const { question, answer, pack } = state.data;

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 px-6 py-10">
      <header>
        <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          сохранённый ответ · read-only · confidence{" "}
          {Math.round(answer.confidence * 100)}%
        </p>
        <h1 className="mt-2 text-lg font-semibold leading-snug text-ink-0">
          {question}
        </h1>
      </header>

      <p className="text-[14px] leading-relaxed text-ink-0">{answer.summary}</p>

      <GuardSeal guard={answer.guard} />

      <section>
        <h2 className="mb-2 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          Методы
        </h2>
        <ul className="flex flex-col gap-1.5">
          {answer.methods.map((method) => (
            <li key={method.name} className="text-[13px] text-ink-1">
              <span className="font-semibold text-ink-0">{method.name}</span> —{" "}
              {method.applicability}{" "}
              <span className="font-mono text-[11px] text-electrolyte">
                [{method.citations.join(", ")}]
              </span>
            </li>
          ))}
        </ul>
      </section>

      <section>
        <h2 className="mb-2 font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          Evidence · {pack.facts.length}
        </h2>
        <div className="flex flex-col gap-2">
          {pack.facts.map((fact) => (
            <div
              key={fact.id}
              className="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-sm border border-line bg-bg-1 px-3 py-2"
            >
              <span className="font-mono text-[11px] font-bold text-electrolyte">
                {fact.ref}
              </span>
              <span className="text-[12px] text-ink-1">
                {fact.parameter.name}
              </span>
              <FactValue value={fact.value} className="text-[12px]" />
              <ValidationBadge status={fact.validationStatus} />
              <span className="ml-auto">
                <ProvenanceStamp
                  provenance={fact.provenance}
                  method={fact.extractionMethod}
                  compact
                />
              </span>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
