"use client";

import { createContext, useContext } from "react";
import { ru } from "./ru";
import { en } from "./en";
import type { Translations } from "./ru";

export type Locale = "ru" | "en";

const translations: Record<Locale, Translations> = { ru, en };

const I18nContext = createContext<{
  t: Translations;
  locale: Locale;
  setLocale: (locale: Locale) => void;
}>({
  t: ru,
  locale: "ru",
  setLocale: () => {},
});

export { I18nContext, translations };

export function useTranslations() {
  const ctx = useContext(I18nContext);
  return ctx.t;
}

export function useLocale() {
  const ctx = useContext(I18nContext);
  return { locale: ctx.locale, setLocale: ctx.setLocale };
}

export function getStoredLocale(): Locale {
  if (typeof window === "undefined") return "ru";
  const stored = localStorage.getItem("locale");
  if (stored === "en" || stored === "ru") return stored;
  return "ru";
}

export function setStoredLocale(locale: Locale): void {
  if (typeof window === "undefined") return;
  localStorage.setItem("locale", locale);
}
