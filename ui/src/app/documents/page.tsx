"use client";

import { Fragment, useEffect, useState } from "react";
import {
  getDocuments,
  openDocumentSource,
  type DocumentRow,
} from "@/shared/api/browse";

const STAGES = ["registered", "parsed", "extracted", "indexed"] as const;

const PAGE_SIZE = 50;

const GEO_LABELS: Record<string, string> = {
  ru: "РФ",
  foreign: "заруб.",
  global: "глоб.",
  unknown: "—",
};

export default function DocumentsPage() {
  const [rows, setRows] = useState<DocumentRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);

  const openSource = async (row: DocumentRow) => {
    const ok = await openDocumentSource(row.id, row.title);
    if (!ok) {
      setToast("Исходный файл недоступен в хранилище");
      setTimeout(() => setToast(null), 3000);
    }
  };

  useEffect(() => {
    let alive = true;
    setLoading(true);
    getDocuments(page * PAGE_SIZE, PAGE_SIZE).then((data) => {
      if (!alive) return;
      setRows(data.items);
      setTotal(data.total);
      setExpanded(null);
      setLoading(false);
    });
    return () => {
      alive = false;
    };
  }, [page]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const from = total === 0 ? 0 : page * PAGE_SIZE + 1;
  const to = page * PAGE_SIZE + rows.length;

  return (
    <div className="mx-auto flex max-w-[1200px] flex-col gap-6 px-6 py-8">
      <section className="rise-in">
        <h1 className="font-display text-xl font-extrabold text-ink-0">
          Реестр документов и конвейер
        </h1>
        <p className="mt-1 text-[13px] text-ink-1">
          Загрузка корпуса и статусы обработки: registered → parsed → extracted →
          indexed
        </p>
        <p className="mt-1 font-mono text-[11px] text-ink-2">
          показано {from}–{to} из {total} документов · список отфильтрован по
          вашему уровню доступа (RLS)
        </p>
      </section>

      <Dropzone />

      <div className="overflow-x-auto rounded-sm border border-line">
        <table className="w-full border-collapse text-[13px]">
          <thead>
            <tr className="border-b border-line bg-bg-1 text-left font-mono text-[10px] uppercase tracking-wider text-ink-2">
              <th className="px-3 py-2">название</th>
              <th className="px-3 py-2">тип</th>
              <th className="px-3 py-2">язык</th>
              <th className="px-3 py-2">гео</th>
              <th className="px-3 py-2">доступ</th>
              <th className="px-3 py-2">конвейер</th>
              <th className="px-3 py-2 text-right">факты</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row, index) => (
              <Fragment key={row.id}>
                <tr
                  onClick={() => setExpanded(expanded === row.id ? null : row.id)}
                  className={`cursor-pointer border-b border-line/60 hover:bg-bg-1 ${index % 2 ? "bg-bg-1/40" : ""}`}
                >
                  <td className="px-3 py-2 text-ink-0">{row.title}</td>
                  <td className="px-3 py-2 font-mono text-[11px] text-ink-1">{row.docType}</td>
                  <td className="px-3 py-2 font-mono text-[11px] text-ink-2">{row.lang}</td>
                  <td className="px-3 py-2 font-mono text-[11px] text-ink-2">
                    {GEO_LABELS[row.geography] ?? row.geography}
                  </td>
                  <td className="px-3 py-2">
                    <span className="rounded-sm border border-line px-1.5 py-0.5 font-mono text-[10px] text-ink-1">
                      {row.accessLevel}
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    <Pipeline status={row.status} />
                  </td>
                  <td className="px-3 py-2 text-right font-mono tabular-nums text-electrolyte">
                    {row.facts}
                  </td>
                </tr>
                {expanded === row.id && (
                  <tr className="border-b border-line/60 bg-bg-0">
                    <td colSpan={7} className="px-3 py-3">
                      <div className="flex flex-wrap items-center gap-6 text-[12px]">
                        <span className="font-mono text-[10px] uppercase tracking-wider text-ink-2">
                          паспорт · {row.id}
                        </span>
                        <span className="text-ink-1">год: {row.year}</span>
                        <span className="text-ink-1">версия: 1</span>
                        <span className="text-ink-1">nc_suspect_rate: 0.02</span>
                        <span className="text-ink-1">llm_valid_rate: 0.94</span>
                        <button
                          type="button"
                          onClick={() => openSource(row)}
                          className="ml-auto rounded-sm border border-line px-2 py-1 font-mono text-[10px] text-ink-1 transition-colors hover:border-electrolyte hover:text-electrolyte"
                        >
                          скачать источник
                        </button>
                        <button
                          type="button"
                          className="rounded-sm border border-line px-2 py-1 font-mono text-[10px] text-ink-1 transition-colors hover:border-electrolyte hover:text-electrolyte"
                        >
                          reindex
                        </button>
                      </div>
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
          </tbody>
        </table>
      </div>

      <Pager
        page={page}
        totalPages={totalPages}
        loading={loading}
        onChange={setPage}
      />

      {toast && (
        <div className="fixed bottom-6 left-6 z-50 rounded-sm border border-anode/50 bg-bg-2 px-4 py-2 text-[12px] text-anode shadow-lg">
          {toast}
        </div>
      )}
    </div>
  );
}

function Pipeline({ status }: { status: string }) {
  const currentIndex = STAGES.indexOf(status as (typeof STAGES)[number]);
  const failed = status === "failed";
  return (
    <div className="flex items-center gap-1" title={status}>
      {STAGES.map((stage, index) => {
        const done = !failed && index <= currentIndex;
        return (
          <span
            key={stage}
            className={`h-2 w-2 rounded-full ${
              failed ? "bg-melt" : done ? "bg-electrolyte" : "bg-bg-2"
            }`}
          />
        );
      })}
    </div>
  );
}

function Pager({
  page,
  totalPages,
  loading,
  onChange,
}: {
  page: number;
  totalPages: number;
  loading: boolean;
  onChange: (page: number) => void;
}) {
  const canPrev = page > 0 && !loading;
  const canNext = page < totalPages - 1 && !loading;
  const buttonClass =
    "rounded-sm border border-line px-2 py-1 font-mono text-[10px] text-ink-1 transition-colors enabled:hover:border-electrolyte enabled:hover:text-electrolyte disabled:cursor-not-allowed disabled:opacity-40";

  return (
    <div className="flex items-center justify-between gap-3">
      <span className="font-mono text-[11px] text-ink-2">
        {loading ? "загрузка…" : `стр. ${page + 1} из ${totalPages}`}
      </span>
      <div className="flex items-center gap-1.5">
        <button
          type="button"
          className={buttonClass}
          disabled={!canPrev}
          onClick={() => onChange(0)}
        >
          « первая
        </button>
        <button
          type="button"
          className={buttonClass}
          disabled={!canPrev}
          onClick={() => onChange(page - 1)}
        >
          ‹ назад
        </button>
        <button
          type="button"
          className={buttonClass}
          disabled={!canNext}
          onClick={() => onChange(page + 1)}
        >
          вперёд ›
        </button>
        <button
          type="button"
          className={buttonClass}
          disabled={!canNext}
          onClick={() => onChange(totalPages - 1)}
        >
          последняя »
        </button>
      </div>
    </div>
  );
}

function Dropzone() {
  return (
    <label className="rise-in flex cursor-pointer flex-col items-center gap-2 rounded-sm border border-dashed border-line-strong bg-bg-1 px-4 py-6 text-center transition-colors hover:border-electrolyte">
      <span className="font-mono text-[11px] uppercase tracking-[0.2em] text-ink-2">
        перетащите файлы или выберите
      </span>
      <span className="text-[12px] text-ink-1">
        PDF · DOCX · TXT · CSV/XLSX · множественная загрузка и манифест каталога
      </span>
      <input
        type="file"
        name="document-upload"
        aria-label="Загрузка документов"
        multiple
        className="hidden"
      />
    </label>
  );
}
