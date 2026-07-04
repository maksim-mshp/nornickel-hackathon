"use client";

import { Fragment, useEffect, useState } from "react";
import { FactValue } from "@/entities/fact/fact-value";
import {
  ProvenanceStamp,
  ValidationBadge,
  docTypeLabel,
} from "@/entities/fact/provenance-stamp";
import type { EvidencePack, Fact, QueryPlan } from "@/shared/api/types";
import { downloadDocumentSource } from "@/shared/lib/download";
import { IconGraph } from "@/shared/ui/icons";
import { EgoGraph } from "@/widgets/ego-graph/ego-graph";

export function Inspector({
  fact,
  plan,
  pack,
}: {
  fact: Fact | null;
  plan: QueryPlan | null;
  pack: EvidencePack | null;
}) {
  const [tab, setTab] = useState<"quote" | "graph">("quote");
  const graphReady = Boolean(plan && pack);

  useEffect(() => {
    if (fact) setTab("quote");
  }, [fact]);

  return (
    <div className="flex h-full flex-col">
      <div className="flex gap-1 border-b border-line px-2 pt-2">
        {(
          [
            { key: "quote", label: "Цитата" },
            { key: "graph", label: "Граф" },
          ] as const
        ).map(({ key, label }) => (
          <button
            key={key}
            type="button"
            onClick={() => setTab(key)}
            className={`border-b-2 px-3 py-1.5 text-[12px] transition-colors ${
              tab === key
                ? "border-electrolyte font-semibold text-ink-0"
                : "border-transparent text-ink-2 hover:text-ink-1"
            }`}
          >
            {label}
          </button>
        ))}
      </div>
      {tab === "quote" ? (
        <QuotePane fact={fact} />
      ) : graphReady ? (
        <div className="p-4">
          <EgoGraph plan={plan!} pack={pack!} />
        </div>
      ) : (
        <EmptyPane text="Граф появится после получения evidence" />
      )}
    </div>
  );
}

function EmptyPane({ text }: { text: string }) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
      <IconGraph className="text-void" width={28} height={28} />
      <p className="text-[12px] text-ink-2">{text}</p>
    </div>
  );
}

function QuotePane({ fact }: { fact: Fact | null }) {
  if (!fact) {
    return (
      <EmptyPane text="Выберите факт в ленте — здесь появится цитата первоисточника и штамп провенанса" />
    );
  }

  return (
    <div className="flex flex-col gap-4 p-4">
      <div>
        <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          цитата · {docTypeLabel(fact.provenance.docType)}
        </span>
        <blockquote className="mt-2 border-l-2 border-electrolyte bg-bg-0 px-3 py-2 font-mono text-[12px] leading-relaxed text-ink-1">
          «<HighlightedQuote quote={fact.provenance.quote} />»
        </blockquote>
      </div>
      <div className="flex flex-col gap-2">
        <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          факт {fact.ref}
        </span>
        <div className="flex items-baseline justify-between gap-2">
          <span className="text-[12px] text-ink-1">{fact.parameter.name}</span>
          <FactValue value={fact.value} className="text-[14px]" />
        </div>
        <div className="flex items-baseline justify-between gap-2">
          <span className="font-mono text-[10px] text-ink-2">SI</span>
          <FactValue value={fact.si} className="text-[11px] opacity-80" />
        </div>
        <ValidationBadge status={fact.validationStatus} />
      </div>
      <div className="flex flex-col gap-2 border-t border-line pt-3">
        <p className="text-[12px] font-semibold leading-snug text-ink-0">
          {fact.provenance.title}
        </p>
        <ProvenanceStamp
          provenance={fact.provenance}
          method={fact.extractionMethod}
        />
        <p className="font-mono text-[10px] text-ink-2">
          {fact.extractorVersion} · confidence{" "}
          {Math.round(fact.confidence * 100)}%
        </p>
        <DownloadSourceButton documentId={fact.provenance.documentId} />
      </div>
    </div>
  );
}

function DownloadSourceButton({ documentId }: { documentId: string }) {
  const [status, setStatus] = useState<"idle" | "loading" | "error">("idle");
  const label =
    status === "loading"
      ? "Загрузка оригинала…"
      : status === "error"
        ? "Оригинал недоступен"
        : "↓ Скачать оригинал";
  return (
    <button
      type="button"
      disabled={status === "loading"}
      onClick={async () => {
        setStatus("loading");
        try {
          await downloadDocumentSource(documentId);
          setStatus("idle");
        } catch {
          setStatus("error");
          setTimeout(() => setStatus("idle"), 3000);
        }
      }}
      className={`mt-1 inline-flex w-fit items-center gap-1.5 rounded-sm border px-2 py-1 font-mono text-[10px] transition-colors disabled:opacity-60 ${
        status === "error"
          ? "border-melt/50 text-melt"
          : "border-line text-ink-1 hover:border-electrolyte hover:text-electrolyte"
      }`}
    >
      {label}
    </button>
  );
}

function HighlightedQuote({ quote }: { quote: string }) {
  const parts = quote.split(/(\d+(?:[.,]\d+)?(?:\s?[–-]\s?\d+(?:[.,]\d+)?)?)/g);
  return (
    <>
      {parts.map((part, index) =>
        index % 2 === 1 ? (
          <mark
            key={index}
            className="bg-electrolyte/15 font-bold text-electrolyte"
          >
            {part}
          </mark>
        ) : (
          <Fragment key={index}>{part}</Fragment>
        ),
      )}
    </>
  );
}
