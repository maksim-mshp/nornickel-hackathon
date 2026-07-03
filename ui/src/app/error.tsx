"use client";

import Link from "next/link";

export default function Error({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="flex min-h-[calc(100vh-56px)] flex-col items-center justify-center gap-6 px-6 text-center">
      <span className="font-display text-6xl font-extrabold tracking-tight text-anode">
        !
      </span>
      <div className="flex flex-col items-center gap-2">
        <h1 className="font-display text-xl font-bold text-ink-0">
          Что-то пошло не так
        </h1>
        <p className="max-w-md text-[14px] text-ink-2">
          Не удалось отобразить этот раздел. Попробуйте повторить или вернуться к
          поиску.
        </p>
      </div>
      <div className="flex gap-3">
        <button
          type="button"
          onClick={reset}
          className="rounded-sm bg-electrolyte px-5 py-2 text-[13px] font-medium text-bg-0 transition-colors hover:bg-electrolyte/90"
        >
          Повторить
        </button>
        <Link
          href="/"
          className="rounded-sm border border-line px-5 py-2 text-[13px] text-ink-1 transition-colors hover:border-line-strong"
        >
          На главную
        </Link>
      </div>
    </div>
  );
}
