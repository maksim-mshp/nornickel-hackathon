"use client";

import { useEffect, useState } from "react";
import { copyShareLink, exportCsv, exportMarkdown } from "@/features/ask/export";
import type { AskState } from "@/features/ask/use-ask";
import type { Fact, QueryPlan } from "@/shared/api/types";
import { pluralCount } from "@/shared/lib/plural";
import { ConsensusSpectrum } from "@/widgets/consensus-spectrum/consensus-spectrum";
import { ContradictionCard } from "./contradiction-card";
import { EvidenceTable } from "./evidence-table";
import { GapsList, ExpertsList } from "./gaps-experts";
import { GuardSeal } from "./guard-seal";
import { PlanChip } from "./plan-chip";
import { MethodsList, SummaryText } from "./summary-view";

type TabKey = "summary" | "evidence" | "contradictions" | "gaps" | "experts";

function intentTab(intent: QueryPlan["intent"] | undefined): TabKey {
  switch (intent) {
    case "expert_search":
      return "experts";
    case "gap_analysis":
      return "gaps";
    case "contradiction_analysis":
      return "contradictions";
    default:
      return "summary";
  }
}

function EmptyTab({ text }: { text: string }) {
  return (
    <p className="py-8 text-center text-[12px] text-ink-2">{text}</p>
  );
}

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
  const [shared, setShared] = useState(false);
  const { plan, pack, answer, summaryText, phase, error, question } = state;

  useEffect(() => {
    if (plan) setTab(intentTab(plan.intent));
  }, [plan]);

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
          <span>
            {pluralCount(pack.stats.sources, "источник", "источника", "источников")}
          </span>
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
                {pack.consensus.map((consensus, index) => (
                  <ConsensusSpectrum
                    key={`${consensus.parameter.slug}-${index}`}
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
            pack.contradictions.length > 0 ? (
              <div className="flex flex-col gap-3">
                {pack.contradictions.map((contradiction) => (
                  <ContradictionCard
                    key={contradiction.id}
                    contradiction={contradiction}
                  />
                ))}
              </div>
            ) : (
              <EmptyTab text="Подтверждённых противоречий по этому запросу не найдено" />
            )
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
            pack.experts.length > 0 ? (
              <ExpertsList experts={pack.experts} />
            ) : (
              <EmptyTab text="Профили экспертов по этому запросу не найдены" />
            )
          ) : (
            <FeedSkeleton lines={3} />
          ))}
      </div>

      {phase === "error" && (
        <div className="flex items-center gap-3 border border-dashed border-melt/60 bg-melt/5 px-4 py-2.5 text-melt">
          <span className="font-mono text-base leading-none">⚠</span>
          <span className="text-[12px]">
            {error ?? "Сервис ответа недоступен"} — факты из evidence остаются
            доступны, синтез можно перезапустить
          </span>
        </div>
      )}

      {answer && (
        <>
          <GuardSeal guard={answer.guard} />
          <div className="flex items-center gap-2">
            <FooterAction
              label="экспорт MD"
              onClick={() => pack && exportMarkdown(question, answer, pack)}
            />
            <FooterAction
              label="экспорт CSV"
              onClick={() => pack && exportCsv(pack)}
            />
            <FooterAction
              label={shared ? "ссылка скопирована ✓" : "поделиться"}
              onClick={async () => {
                if (await copyShareLink(question)) {
                  setShared(true);
                  setTimeout(() => setShared(false), 2500);
                }
              }}
            />
          </div>
        </>
      )}
    </div>
  );
}

function FooterAction({
  label,
  onClick,
}: {
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-sm border border-line px-3 py-1.5 font-mono text-[11px] text-ink-1 transition-colors hover:border-electrolyte hover:text-electrolyte"
    >
      {label}
    </button>
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
