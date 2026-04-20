"use client";

import { ThemeProvider } from "next-themes";
import { Toast } from "@heroui/react";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider attribute={["class", "data-theme"]} defaultTheme="light" enableSystem>
      <Toast.Provider placement="top end" />
      {children}
    </ThemeProvider>
  );
}
