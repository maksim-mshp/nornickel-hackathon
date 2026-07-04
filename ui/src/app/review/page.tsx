"use client";

import { useCallback, useEffect, useState } from "react";
import {
  decideContradiction,
  getContradictions,
  getReviewEntities,
  getReviewOrphans,
  updateEntityStatus,
  updateFactStatus,
  type ContradictionLive,
  type ReviewEntity,
  type ReviewOrphan,
} from "@/shared/api/browse";
import { useRole } from "@/shared/lib/role";

type Tab = "entities" | "orphans" | "contradictions";

const TAB_LABELS: Record<Tab, string> = {
  entities: "Сущности pending",
  orphans: "Числа на ревью",
  contradictions: "Противоречия suspected",
};

export default function ReviewPage() {
  const [tab, setTab] = useState<Tab>("entities");
  const [entities, setEntities] = useState<ReviewEntity[]>([]);
  const [orphans, setOrphans] = useState<ReviewOrphan[]>([]);
  const [contradictions, setContradictions] = useState<ContradictionLive[]>([]);
  const [toast, setToast] = useState<string | null>(null);
  const roleToken = useRole((store) => store.token);

  useEffect(() => {
    let alive = true;
    void Promise.all([
      getReviewEntities(),
      getReviewOrphans(),
      getContradictions("suspected"),
      getContradictions("judge_confirmed"),
    ]).then(([e, o, suspected, judged]) => {
      if (!alive) return;
      setEntities(e);
      setOrphans(o);
      setContradictions([...suspected, ...judged]);
    });
    return () => {
      alive = false;
    };
  }, [roleToken]);

  const counts: Record<Tab, number> = {
    entities: entities.length,
    orphans: orphans.length,
    contradictions: contradictions.length,
  };

  const notify = useCallback((message: string) => {
    setToast(message);
    setTimeout(() => setToast(null), 2500);
  }, []);

  const resolveEntity = useCallback(
    async (id: string, action: string, status: "accept" | "reject") => {
      const ok = await updateEntityStatus(id, status);
      if (!ok) {
        notify("Не удалось сохранить решение — повторите");
        return;
      }
      setEntities((prev) => prev.filter((item) => item.id !== id));
      notify(`Сущность ${action} · audit_log`);
    },
    [notify],
  );

  const resolveOrphan = useCallback(
    async (id: string, action: string, status: string) => {
      const ok = await updateFactStatus(id, status);
      if (!ok) {
        notify("Не удалось сохранить решение — повторите");
        return;
      }
      setOrphans((prev) => prev.filter((item) => item.id !== id));
      notify(`Число ${action} · fact_history`);
    },
    [notify],
  );

  const resolveContradiction = useCallback(
    async (id: string, action: string, decision: "confirmed" | "rejected") => {
      const ok = await decideContradiction(id, decision);
      if (!ok) {
        notify("Не удалось сохранить решение — повторите");
        return;
      }
      setContradictions((prev) => prev.filter((item) => item.id !== id));
      notify(`Противоречие ${action}`);
    },
    [notify],
  );

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey || event.altKey) return;
      const target = event.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === "INPUT" ||
          target.tagName === "TEXTAREA" ||
          target.isContentEditable)
      ) {
        return;
      }
      if (event.key !== "a" && event.key !== "r") return;
      const approve = event.key === "a";
      if (tab === "entities" && entities[0]) {
        resolveEntity(
          entities[0].id,
          approve ? "подтверждена" : "отклонена",
          approve ? "accept" : "reject",
        );
      }
      if (tab === "orphans" && orphans[0]) {
        resolveOrphan(
          orphans[0].id,
          approve ? "подтверждено" : "отклонено",
          approve ? "expert_validated" : "rejected",
        );
      }
      if (tab === "contradictions" && contradictions[0]) {
        resolveContradiction(
          contradictions[0].id,
          approve ? "подтверждено" : "отклонено",
          approve ? "confirmed" : "rejected",
        );
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [tab, entities, orphans, contradictions, resolveEntity, resolveOrphan, resolveContradiction]);

  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-6 px-6 py-8">
      <section className="rise-in">
        <h1 className="font-display text-xl font-extrabold text-ink-0">
          Очередь ревью
        </h1>
        <p className="mt-1 text-[13px] text-ink-1">
          Валидация извлечённого из базы: клавиши{" "}
          <kbd className="rounded-sm border border-line px-1 font-mono text-[11px]">a</kbd>{" "}
          подтвердить ·{" "}
          <kbd className="rounded-sm border border-line px-1 font-mono text-[11px]">r</kbd>{" "}
          отклонить
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
              onApprove={() => resolveEntity(item.id, "подтверждена", "accept")}
              onReject={() => resolveEntity(item.id, "отклонена", "reject")}
            >
              <div className="flex items-baseline gap-2">
                <span className="font-mono text-[10px] uppercase text-ink-2">{item.type}</span>
                <span className="text-[14px] font-semibold text-ink-0">{item.name}</span>
              </div>
              <p className="mt-1 font-mono text-[11px] text-ink-2">{item.slug}</p>
              {item.candidate && (
                <p className="mt-1 text-[12px] text-ink-1">
                  кандидат: {item.candidate}{" "}
                  <span className="font-mono text-[11px] text-ink-2">
                    similarity {item.similarity.toFixed(2)}
                  </span>
                </p>
              )}
            </ReviewCard>
          ))}
        {tab === "orphans" &&
          orphans.map((item) => (
            <ReviewCard
              key={item.id}
              onApprove={() => resolveOrphan(item.id, "подтверждено", "expert_validated")}
              onReject={() => resolveOrphan(item.id, "отклонено", "rejected")}
            >
              <p className="font-mono text-[15px] font-bold text-electrolyte">{item.value}</p>
              {item.quote && <p className="mt-1 text-[12px] italic text-ink-1">«{item.quote}»</p>}
              <div className="mt-2 flex flex-wrap gap-1.5">
                <span className="rounded-sm border border-anode/40 px-1.5 py-0.5 font-mono text-[10px] text-anode">
                  {item.status}
                </span>
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
              onApprove={() => resolveContradiction(item.id, "подтверждено", "confirmed")}
              onReject={() => resolveContradiction(item.id, "отклонено", "rejected")}
            >
              <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-3">
                <p className="text-[12px] text-ink-0">{item.aStatement || item.subject}</p>
                <span className="h-8 w-px bg-melt" />
                <p className="text-[12px] text-ink-0">{item.bStatement || item.parameter}</p>
              </div>
              <p className="mt-2 rounded-sm border border-melt/30 bg-melt/5 px-2 py-1 text-[11px] text-melt">
                судья: {item.cause || `${item.subject} · ${item.parameter}`} · severity{" "}
                {item.severity.toFixed(2)}
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
