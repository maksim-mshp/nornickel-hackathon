"use client";

import { useState } from "react";
import type { AskState } from "@/features/ask/use-ask";
import type { Fact } from "@/shared/api/types";
import { ConsensusSpectrum } from "@/widgets/consensus-spectrum/consensus-spectrum";
import { ContradictionCard } from "./contradiction-card";
import { EvidenceTable } from "./evidence-table";
import { GapsList, ExpertsList } from "./gaps-experts";
import { GuardSeal } from "./guard-seal";
import { PlanChip } from "./plan-chip";
import { MethodsList, SummaryText } from "./summary-view";

type TabKey = "summary" | "evidence" | "contradictions" | "gaps" | "experts";

export function AnswerFeed({
  state,
  selectedFact,
  onSelectFact,
  onAsk,
}: {
  state: AskState;
  selectedFact: Fact | null;
  onSelectFact: (fact: Fact) => void;
  onAsk: (question: string) => void;
}) {
  const [tab, setTab] = useState<TabKey>("summary");
  const { plan, pack, answer, summaryText, phase } = state;

  const selectRef = (ref: string) => {
    const fact = pack?.facts.find((f) => f.ref === ref);
    if (fact) onSelectFact(fact);
  };

  const tabs: { key: TabKey; label: string; count?: number }[] = [
    { key: "summary", label: "Summary" },
    { key: "evidence", label: "Evidence", count: pack?.facts.length },
    {
      key: "contradictions",
      label: "Противоречия",
      count: pack?.contradictions.length,
    },
    { key: "gaps", label: "Пробелы", count: pack?.gaps.length },
    { key: "experts", label: "Эксперты", count: pack?.experts.length },
  ];

  return (
    <div className="flex min-w-0 flex-1 flex-col gap-4">
      {plan ? <PlanChip plan={plan} /> : <PlanSkeleton />}

      {pack && (
        <div
          className="rise-in flex items-center gap-4 font-mono text-[10px] text-ink-2"
          style={{ animationDelay: "40ms" }}
        >
          <span>{pack.stats.sources} источников</span>
          <span>РФ {pack.stats.ruSources}</span>
          <span>заруб. {pack.stats.foreignSources}</span>
          <span>
            {pack.stats.yearFrom}—{pack.stats.yearTo}
          </span>
        </div>
      )}

      <div className="flex gap-1 border-b border-line">
        {tabs.map(({ key, label, count }) => (
          <button
            key={key}
            type="button"
            onClick={() => setTab(key)}
            className={`flex items-center gap-1.5 border-b-2 px-3 py-2 text-[12px] transition-colors ${
              tab === key
                ? "border-electrolyte font-semibold text-ink-0"
                : "border-transparent text-ink-2 hover:text-ink-1"
            }`}
          >
            {label}
            {count !== undefined && (
              <span className="font-mono text-[10px] tabular-nums text-electrolyte">
                {count}
              </span>
            )}
          </button>
        ))}
      </div>

      <div className="min-h-40">
        {tab === "summary" && (
          <div>
            {summaryText ? (
              <SummaryText
                text={summaryText}
                streaming={phase === "streaming"}
                onRefClick={selectRef}
              />
            ) : (
              <FeedSkeleton lines={4} />
            )}
            {answer && <MethodsList answer={answer} onRefClick={selectRef} />}
            {answer && pack && pack.consensus.length > 0 && (
              <div className="mt-4 flex flex-col gap-3">
                <h3 className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
                  Консенсус источников
                </h3>
                {pack.consensus.map((consensus) => (
                  <ConsensusSpectrum
                    key={consensus.parameter.slug}
                    consensus={consensus}
                  />
                ))}
              </div>
            )}
          </div>
        )}
        {tab === "evidence" &&
          (pack ? (
            <EvidenceTable
              facts={pack.facts}
              selectedId={selectedFact?.id ?? null}
              onSelect={onSelectFact}
            />
          ) : (
            <FeedSkeleton lines={6} />
          ))}
        {tab === "contradictions" &&
          (pack ? (
            <div className="flex flex-col gap-3">
              {pack.contradictions.map((contradiction) => (
                <ContradictionCard
                  key={contradiction.id}
                  contradiction={contradiction}
                />
              ))}
            </div>
          ) : (
            <FeedSkeleton lines={3} />
          ))}
        {tab === "gaps" &&
          (pack ? (
            <GapsList gaps={pack.gaps} onAsk={onAsk} />
          ) : (
            <FeedSkeleton lines={3} />
          ))}
        {tab === "experts" &&
          (pack ? (
            <ExpertsList experts={pack.experts} />
          ) : (
            <FeedSkeleton lines={3} />
          ))}
      </div>

      {answer && <GuardSeal guard={answer.guard} />}
    </div>
  );
}

function PlanSkeleton() {
  return (
    <div className="rounded-sm border border-line bg-bg-1 px-4 py-3">
      <div className="flex items-center gap-3">
        <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          разбор запроса…
        </span>
        <span className="h-3 w-32 animate-pulse rounded-sm bg-bg-2" />
      </div>
      <div className="mt-2 flex gap-1.5">
        <span className="h-6 w-24 animate-pulse rounded-sm bg-bg-2" />
        <span className="h-6 w-32 animate-pulse rounded-sm bg-bg-2" />
      </div>
    </div>
  );
}

function FeedSkeleton({ lines }: { lines: number }) {
  return (
    <div className="flex flex-col gap-2">
      {Array.from({ length: lines }, (_, index) => (
        <span
          key={index}
          className="h-4 animate-pulse rounded-sm bg-bg-1"
          style={{ width: `${100 - index * 9}%` }}
        />
      ))}
    </div>
  );
}
