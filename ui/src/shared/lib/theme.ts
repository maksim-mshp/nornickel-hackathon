"use client";

export type ThemeName = "night" | "protocol";

const STORAGE_KEY = "kmap-theme";

export function readTheme(): ThemeName {
  if (typeof window === "undefined") return "night";
  const saved = window.localStorage.getItem(STORAGE_KEY);
  if (saved === "protocol" || saved === "night") return saved;
  return window.matchMedia("(prefers-color-scheme: light)").matches
    ? "protocol"
    : "night";
}

export function applyTheme(theme: ThemeName) {
  document.documentElement.dataset.theme = theme;
  window.localStorage.setItem(STORAGE_KEY, theme);
}

export function toggleTheme(): ThemeName {
  const next: ThemeName = readTheme() === "night" ? "protocol" : "night";
  applyTheme(next);
  return next;
}
