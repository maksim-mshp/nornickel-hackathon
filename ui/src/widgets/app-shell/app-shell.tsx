"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import {
  ROLE_LABELS,
  routeAllowed,
  useRole,
  type DemoRole,
} from "@/shared/lib/role";
import { applyTheme, readTheme, toggleTheme } from "@/shared/lib/theme";
import { CommandPalette } from "@/widgets/command-palette/command-palette";
import {
  IconBook,
  IconCheck,
  IconDocs,
  IconFlask,
  IconGrid,
  IconPeople,
  IconSearch,
  IconTheme,
} from "@/shared/ui/icons";

const NAV_ITEMS = [
  { href: "/", label: "Поиск", icon: IconSearch },
  { href: "/experiments", label: "Эксперименты", icon: IconFlask },
  { href: "/coverage", label: "Покрытие", icon: IconGrid },
  { href: "/experts", label: "Эксперты", icon: IconPeople },
  { href: "/documents", label: "Документы", icon: IconDocs },
  { href: "/review", label: "Ревью", icon: IconCheck },
  { href: "/dictionaries", label: "Словари", icon: IconBook },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const role = useRole((store) => store.role);
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    applyTheme(readTheme());
    setHydrated(true);
  }, []);

  const navItems = NAV_ITEMS.filter(
    ({ href }) => !hydrated || routeAllowed(role, href),
  );

  return (
    <div className="flex h-dvh flex-col">
      <CommandPalette />
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:left-2 focus:top-2 focus:z-50 focus:rounded-sm focus:bg-electrolyte focus:px-3 focus:py-1.5 focus:text-bg-0"
      >
        К содержимому
      </a>
      <Header />
      <div className="flex min-h-0 flex-1">
        <nav
          aria-label="Основная навигация"
          className="flex w-16 shrink-0 flex-col items-center gap-1 border-r border-line bg-bg-1 py-3"
        >
          {navItems.map(({ href, label, icon: Icon }) => {
            const active =
              href === "/" ? pathname === "/" : pathname.startsWith(href);
            return (
              <Link
                key={href}
                href={href}
                title={label}
                aria-current={active ? "page" : undefined}
                className={`flex h-11 w-11 items-center justify-center rounded-sm transition-colors ${
                  active
                    ? "bg-bg-3 text-electrolyte"
                    : "text-ink-2 hover:bg-bg-2 hover:text-ink-0"
                }`}
              >
                <Icon />
              </Link>
            );
          })}
          <div className="mt-auto flex flex-col items-center gap-2 text-[10px] text-ink-2">
            <span
              className="h-2 w-2 rounded-full bg-electrolyte"
              title="Конвейер работает"
            />
            <span className="font-mono">v0.1</span>
          </div>
        </nav>
        <main id="main" className="min-w-0 flex-1 overflow-auto">
          {children}
        </main>
      </div>
    </div>
  );
}

function Header() {
  return (
    <header className="flex h-14 shrink-0 items-center gap-4 border-b border-line bg-bg-1 px-4">
      <Link href="/" className="flex items-center gap-2">
        <span className="stamp-frame flex h-8 w-8 items-center justify-center bg-bg-2 font-display text-sm font-extrabold text-electrolyte">
          k
        </span>
        <span className="font-display text-sm font-bold tracking-wide text-ink-0">
          kmap
        </span>
      </Link>
      <button
        type="button"
        onClick={() => window.dispatchEvent(new Event("kmap:palette"))}
        className="flex h-9 max-w-xl flex-1 items-center gap-2 rounded-sm border border-line bg-bg-0 px-3 text-ink-2 transition-colors hover:border-line-strong hover:text-ink-1"
      >
        <IconSearch width={16} height={16} />
        <span className="flex-1 truncate text-left text-[13px]">
          Спросить или перейти…
        </span>
        <kbd className="rounded-sm border border-line px-1.5 py-0.5 font-mono text-[10px]">
          ⌘K
        </kbd>
      </button>
      <div className="ml-auto flex items-center gap-3">
        <ThemeToggle />
        <RoleSwitcher />
      </div>
    </header>
  );
}

function ThemeToggle() {
  const [, force] = useState(0);
  return (
    <button
      type="button"
      title="Переключить тему"
      onClick={() => {
        toggleTheme();
        force((n) => n + 1);
      }}
      className="flex h-9 w-9 items-center justify-center rounded-sm text-ink-2 transition-colors hover:bg-bg-2 hover:text-ink-0"
    >
      <IconTheme width={18} height={18} />
    </button>
  );
}

function RoleSwitcher() {
  const { role, setRole } = useRole();
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => setHydrated(true), []);
  const shown = hydrated ? role : "admin";

  return (
    <label className="flex items-center gap-2 text-[12px] text-ink-1">
      <span className="stamp-frame flex h-7 w-7 items-center justify-center bg-bg-2 font-mono text-[11px] text-anode">
        {ROLE_LABELS[shown][0]}
      </span>
      <select
        value={shown}
        onChange={(e) => setRole(e.target.value as DemoRole)}
        aria-label="Demo-роль"
        className="rounded-sm border border-line bg-bg-0 px-2 py-1 text-[12px] text-ink-1"
      >
        {(Object.keys(ROLE_LABELS) as DemoRole[]).map((value) => (
          <option key={value} value={value}>
            {ROLE_LABELS[value]}
          </option>
        ))}
      </select>
    </label>
  );
}
