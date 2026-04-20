"use client";

import { useTheme } from "next-themes";
import { useEffect, useState } from "react";
import { Sun, Moon } from "lucide-react";

interface ThemeToggleProps {
  size?: "sm" | "md";
}

const iconSize: Record<string, number> = { sm: 16, md: 20 };

export function ThemeToggle({ size = "md" }: ThemeToggleProps) {
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  useEffect(() => setMounted(true), []);

  if (!mounted) {
    return (
      <div
        className={`${size === "sm" ? "w-8 h-8" : "w-10 h-10"} rounded-xl bg-[var(--glass-bg-elevated)]`}
      />
    );
  }

  const isDark = theme === "dark";

  return (
    <button
      type="button"
      onClick={() => setTheme(isDark ? "light" : "dark")}
      className={`
        ${size === "sm" ? "w-8 h-8" : "w-10 h-10"}
        flex items-center justify-center rounded-xl
        bg-[var(--glass-bg-elevated)] border border-[var(--glass-border-subtle)]
        hover:bg-[var(--accent-subtle)] transition-all duration-200
        cursor-pointer
      `}
      aria-label={isDark ? "Switch to light mode" : "Switch to dark mode"}
    >
      {isDark ? (
        <Sun size={iconSize[size]} className="text-foreground" />
      ) : (
        <Moon size={iconSize[size]} className="text-foreground" />
      )}
    </button>
  );
}
