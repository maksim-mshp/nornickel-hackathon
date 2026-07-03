"use client";

import { usePathname, useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ROLE_ROUTES, useRole } from "@/shared/lib/role";
import { toggleTheme } from "@/shared/lib/theme";
import { PRESETS } from "@/shared/config/presets";
import { IconSearch } from "@/shared/ui/icons";

type Command = {
  id: string;
  group: string;
  label: string;
  hint?: string;
  keywords: string;
  run: () => void;
};

const NAV_LABELS: Record<string, string> = {
  "/": "Поиск",
  "/experiments": "Эксперименты",
  "/coverage": "Покрытие",
  "/experts": "Эксперты",
  "/documents": "Документы",
  "/review": "Ревью",
  "/dictionaries": "Словари",
};

export function CommandPalette() {
  const router = useRouter();
  const pathname = usePathname();
  const role = useRole((store) => store.role);

  const askQuestion = useCallback(
    (question: string) => {
      if (pathname === "/") {
        window.dispatchEvent(
          new CustomEvent("kmap:ask", { detail: question }),
        );
      } else {
        router.push(`/?q=${encodeURIComponent(question)}`);
      }
    },
    [pathname, router],
  );
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [cursor, setCursor] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const close = useCallback(() => {
    setOpen(false);
    setQuery("");
    setCursor(0);
  }, []);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setOpen((prev) => !prev);
      }
      if (event.key === "Escape") close();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [close]);

  useEffect(() => {
    const onOpen = () => setOpen(true);
    window.addEventListener("kmap:palette", onOpen);
    return () => window.removeEventListener("kmap:palette", onOpen);
  }, []);

  useEffect(() => {
    if (open) inputRef.current?.focus();
  }, [open]);

  const commands = useMemo<Command[]>(
    () => [
      ...PRESETS.map((preset) => ({
        id: preset.id,
        group: "Протоколы",
        label: `${preset.code} · ${preset.title}`,
        hint: preset.question,
        keywords: `${preset.code} ${preset.title} ${preset.question}`.toLowerCase(),
        run: () => askQuestion(preset.question),
      })),
      ...ROLE_ROUTES[role].map((href) => ({
        id: `nav${href}`,
        group: "Разделы",
        label: NAV_LABELS[href],
        keywords: `${NAV_LABELS[href]} ${href}`.toLowerCase(),
        run: () => router.push(href),
      })),
      {
        id: "theme",
        group: "Настройки",
        label: "Переключить тему · ночь / протокол",
        keywords: "тема theme ночь протокол светлая тёмная",
        run: () => toggleTheme(),
      },
    ],
    [role, router, askQuestion],
  );

  const trimmed = query.trim();
  const filtered = trimmed
    ? commands.filter((command) =>
        trimmed
          .toLowerCase()
          .split(/\s+/)
          .every((token) => command.keywords.includes(token)),
      )
    : commands;

  const askEntry = trimmed.length > 3;
  const total = filtered.length + (askEntry ? 1 : 0);
  const active = Math.min(cursor, Math.max(total - 1, 0));

  const runAt = (index: number) => {
    if (askEntry && index === 0) {
      askQuestion(trimmed);
    } else {
      filtered[askEntry ? index - 1 : index]?.run();
    }
    close();
  };

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-bg-0/70 pt-[12vh]"
      onClick={close}
      role="dialog"
      aria-modal="true"
      aria-label="Командная палитра"
    >
      <div
        className="w-full max-w-xl rounded-sm border border-line-strong bg-bg-1 shadow-2xl"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-center gap-3 border-b border-line px-4">
          <IconSearch className="shrink-0 text-ink-2" width={16} height={16} />
          <input
            ref={inputRef}
            value={query}
            onChange={(event) => {
              setQuery(event.target.value);
              setCursor(0);
            }}
            onKeyDown={(event) => {
              if (event.key === "ArrowDown") {
                event.preventDefault();
                setCursor((prev) => Math.min(prev + 1, total - 1));
              }
              if (event.key === "ArrowUp") {
                event.preventDefault();
                setCursor((prev) => Math.max(prev - 1, 0));
              }
              if (event.key === "Enter" && total > 0) {
                event.preventDefault();
                runAt(active);
              }
            }}
            placeholder="Спросить или перейти…"
            className="h-12 flex-1 bg-transparent text-[14px] text-ink-0 placeholder:text-ink-2 focus:outline-none"
          />
          <kbd className="rounded-sm border border-line px-1.5 py-0.5 font-mono text-[10px] text-ink-2">
            esc
          </kbd>
        </div>
        <div className="max-h-[50vh] overflow-auto p-2">
          {askEntry && (
            <PaletteRow
              active={active === 0}
              group="Запрос"
              label={`Спросить: «${trimmed}»`}
              onClick={() => runAt(0)}
            />
          )}
          {filtered.map((command, index) => (
            <PaletteRow
              key={command.id}
              active={active === (askEntry ? index + 1 : index)}
              group={command.group}
              label={command.label}
              hint={command.hint}
              onClick={() => runAt(askEntry ? index + 1 : index)}
            />
          ))}
          {total === 0 && (
            <p className="px-3 py-4 text-[12px] text-ink-2">
              Ничего не найдено — Enter выполнит это как вопрос
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

function PaletteRow({
  active,
  group,
  label,
  hint,
  onClick,
}: {
  active: boolean;
  group: string;
  label: string;
  hint?: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex w-full items-baseline gap-3 rounded-sm px-3 py-2 text-left transition-colors ${
        active ? "bg-bg-2" : "hover:bg-bg-2/60"
      }`}
    >
      <span className="w-20 shrink-0 font-mono text-[9px] uppercase tracking-wider text-ink-2">
        {group}
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-[13px] text-ink-0">{label}</span>
        {hint && (
          <span className="block truncate text-[11px] text-ink-2">{hint}</span>
        )}
      </span>
    </button>
  );
}
