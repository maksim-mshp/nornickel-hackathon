"use client";

export type ThemeName = "night" | "protocol";

const STORAGE_KEY = "kmap-theme";

function safeGet(key: string): string | null {
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
}

function safeSet(key: string, value: string): void {
  try {
    window.localStorage.setItem(key, value);
  } catch {
    return;
  }
}

export function readTheme(): ThemeName {
  if (typeof window === "undefined") return "night";
  const saved = safeGet(STORAGE_KEY);
  if (saved === "protocol" || saved === "night") return saved;
  return window.matchMedia("(prefers-color-scheme: light)").matches
    ? "protocol"
    : "night";
}

export function applyTheme(theme: ThemeName) {
  document.documentElement.dataset.theme = theme;
  safeSet(STORAGE_KEY, theme);
}

export function toggleTheme(): ThemeName {
  const next: ThemeName = readTheme() === "night" ? "protocol" : "night";
  applyTheme(next);
  return next;
}
