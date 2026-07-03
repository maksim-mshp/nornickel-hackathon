import Link from "next/link";

export default function NotFound() {
  return (
    <div className="flex min-h-[calc(100vh-56px)] flex-col items-center justify-center gap-6 px-6 text-center">
      <span className="font-display text-7xl font-extrabold tracking-tight text-electrolyte">
        404
      </span>
      <div className="flex flex-col items-center gap-2">
        <h1 className="font-display text-xl font-bold text-ink-0">
          Страница не найдена
        </h1>
        <p className="max-w-md text-[14px] text-ink-2">
          Такого маршрута нет в карте знаний. Проверьте адрес или вернитесь к
          поиску.
        </p>
      </div>
      <Link
        href="/"
        className="rounded-sm bg-electrolyte px-5 py-2 text-[13px] font-medium text-bg-0 transition-colors hover:bg-electrolyte/90"
      >
        На главную
      </Link>
    </div>
  );
}
