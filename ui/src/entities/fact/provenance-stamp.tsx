import type { Provenance, ValidationStatus } from "@/shared/api/types";

const METHOD_LABELS: Record<string, string> = {
  deterministic: "numcore",
  llm: "LLM",
  hybrid: "гибрид",
  catalog: "каталог",
};

const DOC_TYPE_LABELS: Record<string, string> = {
  report: "отчёт",
  protocol: "протокол",
  article: "статья",
  patent: "патент",
  thesis: "диссертация",
};

export function ProvenanceStamp({
  provenance,
  method,
  onClick,
  compact = false,
}: {
  provenance: Provenance;
  method?: string;
  onClick?: () => void;
  compact?: boolean;
}) {
  const Tag = onClick ? "button" : "div";
  return (
    <Tag
      type={onClick ? "button" : undefined}
      onClick={onClick}
      className={`stamp-frame inline-flex items-center gap-2 bg-bg-0 px-2 py-1 text-left font-mono text-[10px] leading-tight text-ink-1 ${
        onClick
          ? "cursor-pointer transition-colors hover:border-electrolyte hover:text-ink-0"
          : ""
      }`}
      title={provenance.title}
    >
      <span className="font-bold text-electrolyte">
        № {provenance.documentId.replace("doc_", "")}
      </span>
      <span className="text-ink-2">·</span>
      <span>стр. {provenance.page}</span>
      <span className="text-ink-2">·</span>
      <span>{provenance.year}</span>
      {!compact && method && (
        <>
          <span className="text-ink-2">·</span>
          <span className="uppercase tracking-wide text-ink-2">
            {METHOD_LABELS[method] ?? method}
          </span>
        </>
      )}
    </Tag>
  );
}

export function docTypeLabel(docType: string): string {
  return DOC_TYPE_LABELS[docType] ?? docType;
}

const STATUS_META: Record<
  ValidationStatus,
  { label: string; className: string }
> = {
  machine_extracted: { label: "машинное", className: "text-ink-2 border-line" },
  weak_evidence: {
    label: "слабая база",
    className: "text-void border-line",
  },
  multi_source: {
    label: "мультиисточник",
    className: "text-electrolyte border-electrolyte/40",
  },
  expert_validated: {
    label: "эксперт ✓",
    className: "text-anode border-anode/40",
  },
  contradicted: {
    label: "противоречие",
    className: "text-melt border-melt/40",
  },
};

export function ValidationBadge({ status }: { status: ValidationStatus }) {
  const meta = STATUS_META[status];
  return (
    <span
      className={`inline-flex items-center whitespace-nowrap rounded-sm border px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-wide ${meta.className}`}
    >
      {meta.label}
    </span>
  );
}
