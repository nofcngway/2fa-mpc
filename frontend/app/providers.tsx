"use client";

import { useState, useEffect } from "react";
import { ThemeProvider } from "next-themes";
import { Toast } from "@heroui/react";
import {
  I18nContext,
  translations,
  getStoredLocale,
  setStoredLocale,
  type Locale,
} from "@/lib/i18n";

export function Providers({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("ru");
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setLocaleState(getStoredLocale());
    setMounted(true);
  }, []);

  const setLocale = (l: Locale) => {
    setLocaleState(l);
    setStoredLocale(l);
  };

  // Prevent hydration mismatch — render default locale on server
  const currentLocale = mounted ? locale : "ru";

  return (
    <ThemeProvider attribute={["class", "data-theme"]} defaultTheme="light" enableSystem>
      <I18nContext value={{ t: translations[currentLocale], locale: currentLocale, setLocale }}>
        <Toast.Provider placement="top end" />
        {children}
      </I18nContext>
    </ThemeProvider>
  );
}
