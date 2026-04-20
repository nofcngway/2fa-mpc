"use client";

import { useLocale, useTranslations } from "@/lib/i18n";

interface LocaleToggleProps {
  size?: "sm" | "md";
}

export function LocaleToggle({ size = "md" }: LocaleToggleProps) {
  const { locale, setLocale } = useLocale();
  const t = useTranslations();

  const next = locale === "ru" ? "en" : "ru";

  return (
    <button
      type="button"
      onClick={() => setLocale(next)}
      className={`
        ${size === "sm" ? "h-8 px-2 text-xs" : "h-10 px-3 text-sm"}
        inline-flex items-center gap-1.5 rounded-xl
        bg-[var(--glass-bg-elevated)] border border-[var(--glass-border-subtle)]
        hover:bg-[var(--accent-subtle)] transition-all duration-200
        cursor-pointer font-medium
      `}
      aria-label={t.lang[next]}
    >
      <span className={size === "sm" ? "text-sm" : "text-base"}>{locale === "ru" ? "🇷🇺" : "🇺🇸"}</span>
      <span className="uppercase">{locale}</span>
    </button>
  );
}
