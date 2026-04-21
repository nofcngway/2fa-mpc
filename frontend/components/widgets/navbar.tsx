"use client";

import { Logo } from "@/components/ui/logo";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import { LocaleToggle } from "@/components/ui/locale-toggle";
import { GlassButton } from "@/components/ui/glass-button";
import { useAuth } from "@/hooks/use-auth";
import { useTranslations } from "@/lib/i18n";
import { LogOut } from "lucide-react";

export function Navbar() {
  const { user, logout } = useAuth();
  const t = useTranslations();

  return (
    <header className="absolute top-0 left-0 right-0 z-50">
      <div className="flex items-center justify-between px-6 py-4">
        <Logo size="sm" />

        <div className="flex items-center gap-3">
          {user && (
            <span className="text-sm text-muted hidden sm:block">
              {user.email}
            </span>
          )}
          <LocaleToggle size="sm" />
          <ThemeToggle size="sm" />
          <GlassButton
            variant="ghost"
            size="sm"
            onPress={logout}
            icon={<LogOut size={16} />}
            aria-label={t.navbar.logout}
          >
            <span className="hidden sm:inline">{t.navbar.logout}</span>
          </GlassButton>
        </div>
      </div>
    </header>
  );
}
