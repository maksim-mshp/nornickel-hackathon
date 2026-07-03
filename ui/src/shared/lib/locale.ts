"use client";

export const LOCALE_COOKIE = "kmap_locale";
export const LOCALES = ["ru", "en"] as const;
export type Locale = (typeof LOCALES)[number];

export function setLocale(locale: Locale): void {
  document.cookie = `${LOCALE_COOKIE}=${locale}; path=/; max-age=31536000; samesite=lax`;
  window.location.reload();
}
