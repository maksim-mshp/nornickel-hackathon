"use client";

import { useEffect, useState } from "react";
import {
  decideContradiction,
  getContradictions,
  type ContradictionLive,
} from "@/shared/api/browse";

type Tab = "entities" | "orphans" | "contradictions";

type EntityItem = { id: string; name: string; type: string; candidate: string; similarity: number };
type OrphanItem = { id: string; quote: string; value: string; candidates: string[] };
type ContradictionItem = {
  id: string;
  a: string;
  b: string;
  cause: string;
  confidence: number;
};

const ENTITIES: EntityItem[] = [
  { id: "e1", name: "циркуляция католита", type: "process", candidate: "циркуляция католита (process:catholyte-circulation)", similarity: 0.72 },
  { id: "e2", name: "ПВП", type: "equipment", candidate: "печь взвешенной плавки (equipment:flash-furnace)", similarity: 0.58 },
  { id: "e3", name: "сухой остаток", type: "property", candidate: "сухой остаток (property:tds)", similarity: 0.91 },
];

const ORPHANS: OrphanItem[] = [
  { id: "o1", value: "250 А/м²", quote: "плотность тока поддерживалась на уровне 250 А/м²", candidates: ["parameter:current-density", "parameter:flow-rate"] },
  { id: "o2", value: "1,2 г/л", quote: "концентрация примесей не превышала 1,2 г/л", candidates: ["parameter:impurity-concentration"] },
];

const CONTRADICTIONS: ContradictionItem[] = [
  {
    id: "c1",
    a: "0,8–1,0 м/с улучшает равномерность осаждения [F1]",
    b: "выше 0,7 м/с растёт дефектность осадка [F2]",
    cause: "различие плотности тока: 220 vs 320 А/м²",
    confidence: 0.86,
  },
];

const TAB_LABELS: Record<Tab, string> = {
  entities: "Сущности pending",
  orphans: "Числа-сироты",
  contradictions: "Противоречия suspected",
};

function toReviewItem(item: ContradictionLive): ContradictionItem {
  return {
    id: item.id,
    a: item.aStatement || item.subject,
    b: item.bStatement || item.parameter,
    cause: item.cause || `${item.subject} · ${item.parameter}`.trim(),
    confidence: item.severity,
  };
}

export default function ReviewPage() {
  const [tab, setTab] = useState<Tab>("entities");
  const [entities, setEntities] = useState(ENTITIES);
  const [orphans, setOrphans] = useState(ORPHANS);
  const [contradictions, setContradictions] = useState(CONTRADICTIONS);
  const [live, setLive] = useState(false);
  const [toast, setToast] = useState<string | null>(null);

  useEffect(() => {
    let alive = true;
    getContradictions().then((items) => {
      if (!alive || items.length === 0) return;
      setContradictions(items.map(toReviewItem));
      setLive(true);
    });
    return () => {
      alive = false;
    };
  }, []);

  const counts: Record<Tab, number> = {
    entities: entities.length,
    orphans: orphans.length,
    contradictions: contradictions.length,
  };

  const notify = (message: string) => {
    setToast(message);
    setTimeout(() => setToast(null), 2500);
  };

  const resolveEntity = (id: string, action: string) => {
    setEntities((prev) => prev.filter((item) => item.id !== id));
    notify(`Сущность ${action} · записано в fact_history`);
  };
  const resolveOrphan = (id: string, action: string) => {
    setOrphans((prev) => prev.filter((item) => item.id !== id));
    notify(`Число ${action}`);
  };
  const resolveContradiction = (id: string, action: string) => {
    setContradictions((prev) => prev.filter((item) => item.id !== id));
    notify(`Противоречие ${action}`);
    if (live) {
      void decideContradiction(id, action === "подтверждено" ? "confirmed" : "rejected");
    }
  };

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if (event.key === "a" || event.key === "r") {
        const action = event.key === "a" ? "подтверждено" : "отклонено";
        if (tab === "entities" && entities[0]) resolveEntity(entities[0].id, action);
        if (tab === "orphans" && orphans[0]) resolveOrphan(orphans[0].id, action);
        if (tab === "contradictions" && contradictions[0])
          resolveContradiction(contradictions[0].id, action);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [tab, entities, orphans, contradictions]);

  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-6 px-6 py-8">
      <section className="rise-in">
        <h1 className="font-display text-xl font-extrabold text-ink-0">
          Очередь ревью
        </h1>
        <p className="mt-1 text-[13px] text-ink-1">
          Валидация извлечённого: клавиши{" "}
          <kbd className="rounded-sm border border-line px-1 font-mono text-[11px]">a</kbd>{" "}
          подтвердить ·{" "}
          <kbd className="rounded-sm border border-line px-1 font-mono text-[11px]">r</kbd>{" "}
          отклонить
          {live ? " · противоречия из базы" : ""}
        </p>
      </section>

      <div className="flex gap-1 border-b border-line">
        {(Object.keys(TAB_LABELS) as Tab[]).map((key) => (
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
            {TAB_LABELS[key]}
            <span className="font-mono text-[10px] tabular-nums text-electrolyte">
              {counts[key]}
            </span>
          </button>
        ))}
      </div>

      <div className="flex flex-col gap-3">
        {tab === "entities" &&
          entities.map((item) => (
            <ReviewCard
              key={item.id}
              onApprove={() => resolveEntity(item.id, "подтверждено")}
              onReject={() => resolveEntity(item.id, "отклонено")}
            >
              <div className="flex items-baseline gap-2">
                <span className="font-mono text-[10px] uppercase text-ink-2">{item.type}</span>
                <span className="text-[14px] font-semibold text-ink-0">{item.name}</span>
              </div>
              <p className="mt-1 text-[12px] text-ink-1">
                кандидат: {item.candidate}
              </p>
              <p className="mt-1 font-mono text-[11px] text-ink-2">
                similarity {item.similarity.toFixed(2)}
              </p>
            </ReviewCard>
          ))}
        {tab === "orphans" &&
          orphans.map((item) => (
            <ReviewCard
              key={item.id}
              onApprove={() => resolveOrphan(item.id, "привязано")}
              onReject={() => resolveOrphan(item.id, "отклонено")}
            >
              <p className="font-mono text-[15px] font-bold text-electrolyte">{item.value}</p>
              <p className="mt-1 text-[12px] italic text-ink-1">«{item.quote}»</p>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {item.candidates.map((candidate) => (
                  <span
                    key={candidate}
                    className="rounded-sm border border-line px-1.5 py-0.5 font-mono text-[10px] text-ink-1"
                  >
                    {candidate}
                  </span>
                ))}
              </div>
            </ReviewCard>
          ))}
        {tab === "contradictions" &&
          contradictions.map((item) => (
            <ReviewCard
              key={item.id}
              onApprove={() => resolveContradiction(item.id, "подтверждено")}
              onReject={() => resolveContradiction(item.id, "отклонено")}
            >
              <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-3">
                <p className="text-[12px] text-ink-0">{item.a}</p>
                <span className="h-8 w-px bg-melt" />
                <p className="text-[12px] text-ink-0">{item.b}</p>
              </div>
              <p className="mt-2 rounded-sm border border-melt/30 bg-melt/5 px-2 py-1 text-[11px] text-melt">
                судья: {item.cause} · confidence {item.confidence.toFixed(2)}
              </p>
            </ReviewCard>
          ))}
        {counts[tab] === 0 && (
          <p className="py-10 text-center text-[13px] text-ink-2">
            Очередь пуста — всё проверено
          </p>
        )}
      </div>

      {toast && (
        <div className="fixed bottom-6 left-6 z-50 rounded-sm border border-line-strong bg-bg-2 px-4 py-2 text-[12px] text-ink-0 shadow-lg">
          {toast}
        </div>
      )}
    </div>
  );
}

function ReviewCard({
  children,
  onApprove,
  onReject,
}: {
  children: React.ReactNode;
  onApprove: () => void;
  onReject: () => void;
}) {
  return (
    <article className="rise-in flex items-center gap-4 rounded-sm border border-line bg-bg-1 p-4">
      <div className="min-w-0 flex-1">{children}</div>
      <div className="flex shrink-0 flex-col gap-2">
        <button
          type="button"
          onClick={onApprove}
          className="rounded-sm bg-electrolyte px-3 py-1.5 font-mono text-[11px] font-medium text-bg-0 transition-colors hover:bg-electrolyte/90"
        >
          подтвердить
        </button>
        <button
          type="button"
          onClick={onReject}
          className="rounded-sm border border-melt/50 px-3 py-1.5 font-mono text-[11px] text-melt transition-colors hover:bg-melt/10"
        >
          отклонить
        </button>
      </div>
    </article>
  );
}
